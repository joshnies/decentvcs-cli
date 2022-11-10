package storage

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/util"
	"github.com/decentvcs/cli/models"
	"github.com/gammazero/workerpool"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/maps"
)

type AdditionalPresignData struct {
	Multipart          bool
	FileSize           int64
	ContentType        string
	CompressedFilePath string
}

// Upload many objects to storage.
//
// Params:
//
// - projectConfig: Project config
//
// - hashMap: Map of local file paths to file hashes (which are used as object keys)
func UploadMany(projectConfig models.ProjectConfig, hashMap map[string]string) error {
	auth.HasToken()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Presign objects in chunks
	// This is done in chunks to avoid Stytch rate limiting due to the sheer amount of authentication requests
	console.Info("Getting things ready...")
	hashMapChunked := util.ChunkMap(hashMap, config.I.VCS.Storage.PresignChunkSize)
	presignRes := make(map[string]models.PresignResponse)    // map of file path to presign response
	additionalData := make(map[string]AdditionalPresignData) // map of file path to additional data
	for chunkIdx, hashMapChunk := range hashMapChunked {
		console.Verbose("Presigning chunk %d/%d...", chunkIdx+1, len(hashMapChunked))
		bodyData := make(map[string]models.PresignOneRequest) // map of file path to req body data

		for filePath, hash := range hashMapChunk {
			// Open file
			file, err := os.Open(filePath)
			if err != nil {
				panic(console.Error("Failed to open file \"%s\": %v", filePath, err))
			}

			// Get file size
			fileInfo, err := file.Stat()
			if err != nil {
				panic(console.Error("Failed to get file info for file \"%s\": %v", filePath, err))
			}
			fileSize := fileInfo.Size()
			multipart := fileSize > config.I.VCS.Storage.PartSize

			// TODO: Detect MIME type
			// Get MIME content type
			// var contentType string
			// mtype, err := mimetype.DetectReader(file)
			// if err != nil {
			// 	contentType = "application/octet-stream"
			// 	console.Warning("Failed to detect MIME type for file \"%s\", using default \"%s\"", params.FilePath, contentType)
			// } else {
			// 	contentType = mtype.String()
			// }

			contentType := "application/octet-stream"
			var compressedFilePath string

			if multipart {
				// Compress file
				tempDir := os.TempDir()
				compressedFile, err := os.CreateTemp(tempDir, filepath.Base(filePath)+".tmp-")
				if err != nil {
					panic(console.Error("Failed to open temp file for compression: %v", err))
				}
				compressedFilePath := compressedFile.Name()
				console.Verbose("[%s] Compressing; temp file: \"%s\"...", hash, compressedFilePath)
				err = Compress(file, compressedFile)
				if err != nil {
					panic(console.Error("Failed to compress file \"%s\": %v", filePath, err))
				}
				defer compressedFile.Close()

				// Stat compressed file to get file size
				compressedFileInfo, err := compressedFile.Stat()
				if err != nil {
					panic(console.Error("Failed to get file info for file \"%s\": %v", filePath, err))
				}
				fileSize = compressedFileInfo.Size()
			}

			// Get presigned URL for uploading the object later
			bodyData[filePath] = models.PresignOneRequest{
				Method:      "PUT",
				Key:         hash,
				ContentType: contentType,
				Multipart:   multipart,
				Size:        fileSize,
			}

			// Save additional data calculated above
			// This is used to prevent fetching this information again later (performance reasons)
			additionalData[filePath] = AdditionalPresignData{
				Multipart:          multipart,
				FileSize:           fileSize,
				ContentType:        contentType,
				CompressedFilePath: compressedFilePath,
			}
		}

		bodyJson, err := json.Marshal(maps.Values(bodyData))
		if err != nil {
			panic(console.Error("Error marshalling presign request body: %v", err))
		}

		httpClient := http.Client{}
		reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/many", config.I.VCS.ServerHost, projectConfig.ProjectSlug)
		req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
		if err != nil {
			panic(err)
		}
		req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
		req.Header.Add("Content-Type", "application/json")
		res, err := httpClient.Do(req)
		if err != nil {
			panic(console.Error("Error presigning files: %v", err))
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			panic(console.Error("Error presigning files: %v", err))
		}
		defer res.Body.Close()

		// Parse response
		var newRes map[string]models.PresignResponse
		err = json.NewDecoder(res.Body).Decode(&newRes)
		if err != nil {
			panic(console.Error("Error parsing presign response: %v", err))
		}

		presignRes = util.MergeMaps(presignRes, newRes)
	}

	startTime := time.Now()
	bar := progressbar.Default(int64(len(hashMap)))

	// Upload objects in parallel
	console.Info("Uploading...")
	var wg sync.WaitGroup
	for hash, presignRes := range presignRes {
		wg.Add(1)

		uncompressedPath := util.ReverseLookup(hashMap, hash)
		ad := additionalData[uncompressedPath]

		// Determine whether to upload compressed or uncompressed file (single vs multipart)
		var path string
		if ad.CompressedFilePath == "" {
			path = uncompressedPath
		} else {
			path = ad.CompressedFilePath
		}

		// Upload
		go upload(ctx, uploadParams{
			UploadID:      presignRes.UploadID,
			URLs:          presignRes.URLs,
			ProjectConfig: projectConfig,
			FilePath:      path,
			ContentType:   ad.ContentType,
			Multipart:     ad.Multipart,
			Size:          ad.FileSize,
			Hash:          hash,
			Bar:           bar,
			WG:            &wg,
		})
	}

	// Wait for all uploads to finish
	wg.Wait()

	endTime := time.Now()
	console.Info("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type uploadParams struct {
	UploadID      string
	URLs          []string
	ProjectConfig models.ProjectConfig
	FilePath      string
	ContentType   string
	Multipart     bool
	Size          int64
	Hash          string
	Bar           *progressbar.ProgressBar
	WG            *sync.WaitGroup
}

// Upload object to storage. Can be multipart or in full.
// Intended to be called as a goroutine.
func upload(ctx context.Context, params uploadParams) {
	defer params.WG.Done()
	defer params.Bar.Add(1)

	// Read file into byte array
	fileBytes, err := os.ReadFile(params.FilePath)
	if err != nil {
		panic(console.Error("Failed to read file \"%s\": %v", params.FilePath, err))
	}

	if params.Multipart {
		// Delete local compressed file since it's no longer needed
		err = os.Remove(params.FilePath)
		if err != nil {
			panic(console.Error("Failed to delete temp compressed file \"%s\": %v", params.FilePath, err))
		}

		// Upload as multipart
		console.Verbose("[%s] Uploading (multipart)...", params.Hash)
		uploadMultipart(ctx, params, fileBytes)
	} else {
		// Upload in full
		console.Verbose("[%s] Uploading (single)...", params.Hash)
		uploadSingle(ctx, params, fileBytes)
	}

	additionalStorageUsedMB := float64(params.Size) / 1024 / 1024

	// Update team usage
	httpClient := http.Client{}
	teamName := strings.Split(params.ProjectConfig.ProjectSlug, "/")[0]
	reqUrl := fmt.Sprintf("%s/teams/%s/usage", config.I.VCS.ServerHost, teamName)
	reqData := models.UpdateTeamRequest{
		StorageUsedMB: additionalStorageUsedMB, // this is the additional storage used by this upload in MB
	}
	reqJSON, _ := json.Marshal(reqData)
	req, err := http.NewRequest("PUT", reqUrl, bytes.NewBuffer(reqJSON))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(err)
	}
	defer res.Body.Close()
}

// Upload object in full to storage.
// Intended to be called as a goroutine.
func uploadSingle(ctx context.Context, params uploadParams, fileBytes []byte) {
	attempt := 0

	for {
		if attempt >= config.I.VCS.Storage.MaxUploadAttempts {
			panic(console.Error("Failed to upload file \"%s\" after %d attempts", params.FilePath, attempt))
		}

		// Wait until rate limiter frees up before uploading to storage
		err := config.I.RateLimiter.Wait(context.Background())
		if err != nil {
			panic(err)
		}

		// Upload using presigned URL
		console.Verbose("[%s] Uploading...", params.Hash)
		url := params.URLs[0]
		var httpClient http.Client
		req, err := http.NewRequest("PUT", url, bytes.NewReader(fileBytes))
		if err != nil {
			panic(err)
		}
		req.Header.Add("Content-Type", params.ContentType)
		res, err := httpClient.Do(req)
		if err != nil {
			panic(console.Error("Error uploading file \"%s\":\n%v", params.FilePath, err))
		}
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusServiceUnavailable || res.StatusCode == http.StatusForbidden {
			// Rate limited by storage provider, retry after delay
			console.Verbose("[%s] Rate limited; retrying after %ds...", params.Hash, config.I.VCS.Storage.RateLimitRetryDelay)
			attempt++
			time.Sleep(time.Duration(config.I.VCS.Storage.RateLimitRetryDelay) * time.Second)
			continue
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			panic(console.Error("Error uploading file \"%s\": %v", params.FilePath, err))
		}
		res.Body.Close() // close immediately since we dont need it

		console.Verbose("[%s] Uploaded", params.Hash)
		break
	}
}

// Upload a file in chunks to storage.
// Intended to be called as a goroutine.
func uploadMultipart(ctx context.Context, params uploadParams, fileBytes []byte) {
	// Split file into chunks
	chunks := [][]byte{}
	var start int64
	remaining := int64(len(fileBytes))
	for remaining > 0 {
		chunkSize := int64(math.Min(float64(remaining), float64(config.I.VCS.Storage.PartSize)))
		chunks = append(chunks, fileBytes[start:start+chunkSize])
		start += chunkSize
		remaining -= chunkSize
	}

	// Upload parts in sequence.
	var wg sync.WaitGroup
	ch := make(chan models.MultipartUploadPart)
	parts := []models.MultipartUploadPart{}
	totalParts := len(chunks)
	for i, url := range params.URLs {
		wg.Add(1)
		partNum := i + 1
		go uploadPart(ctx, uploadPartParams{
			ProjectID:   params.ProjectConfig.ProjectSlug,
			URL:         url,
			Hash:        params.Hash,
			ContentType: params.ContentType,
			PartNumber:  partNum,
			PartData:    chunks[i],
			TotalParts:  totalParts,
			WG:          &wg,
			Channel:     ch,
		})
		p := <-ch
		parts = append(parts, p)
	}

	wg.Wait()

	// Wait until rate limiter frees up before completing the upload
	err := config.I.RateLimiter.Wait(context.Background())
	if err != nil {
		panic(err)
	}

	// Complete multipart upload
	console.Verbose("[%s] Completing...", params.Hash)
	complBodyData := models.CompleteMultipartUploadRequestBody{
		UploadId: params.UploadID,
		Key:      params.Hash,
		Parts:    parts,
	}
	complBodyJson, err := json.Marshal(complBodyData)
	if err != nil {
		panic(console.Error("Error marshalling \"complete multipart upload\" request body for file \"%s\": %v", params.FilePath, err))
	}

	var httpClient http.Client
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/multipart/complete", config.I.VCS.ServerHost, params.ProjectConfig.ProjectSlug)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(complBodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		panic(console.Error("Error completing multipart upload for file \"%s\": %v", params.FilePath, err))
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(console.Error("Error completing multipart upload for file \"%s\": %v", params.FilePath, err))
	}
	res.Body.Close() // close immediately since we dont need it
	console.Verbose("[%s] Complete", params.Hash)
}

type uploadPartParams struct {
	ProjectID   string
	URL         string
	Hash        string
	ContentType string
	PartNumber  int
	PartData    []byte
	TotalParts  int
	WG          *sync.WaitGroup
	Channel     chan models.MultipartUploadPart
}

// Upload part to storage for a multipart upload.
func uploadPart(ctx context.Context, params uploadPartParams) {
	defer params.WG.Done()

	console.Verbose("[%s] (Part %d/%d) Uploading...", params.Hash, params.PartNumber, params.TotalParts)

	attempt := 0

	for {
		if attempt >= config.I.VCS.Storage.MaxUploadAttempts {
			panic(console.Error("Failed to upload part %d of file \"%s\" after %d attempts", params.PartNumber, params.Hash, attempt))
		}

		// Wait until rate limiter frees up before uploading to storage
		err := config.I.RateLimiter.Wait(context.Background())
		if err != nil {
			panic(err)
		}

		// Upload part
		httpClient := http.Client{}
		req, err := http.NewRequest("PUT", params.URL, bytes.NewReader(params.PartData))
		if err != nil {
			panic(err)
		}
		req.Header.Add("Content-Type", params.ContentType)
		res, err := httpClient.Do(req)
		if err != nil {
			panic(console.Error("[%s] (Part %d/%d) Error uploading part: %v", params.Hash, params.PartNumber, params.TotalParts, err))
		}
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusServiceUnavailable || res.StatusCode == http.StatusForbidden {
			// Rate limited by storage provider, retry after delay
			console.Verbose("[%s] (Part %d/%d) Rate limited; retrying after %ds...", params.Hash, params.PartNumber, params.TotalParts, config.I.VCS.Storage.RateLimitRetryDelay)
			attempt++
			time.Sleep(time.Duration(config.I.VCS.Storage.RateLimitRetryDelay) * time.Second)
			continue
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			panic(console.Error("[%s] (Part %d/%d) Error uploading part: %v", params.Hash, params.PartNumber, params.TotalParts, err))
		}
		res.Body.Close() // close immediately since we dont need it

		// Validate response headers
		etag := res.Header.Get("etag")
		if etag == "" {
			panic(console.Error("[%s] No \"etag\" header returned for part %d", params.Hash, params.PartNumber))
		}

		console.Verbose("[%s] (Part %d/%d) Uploaded; ETag: %s", params.Hash, params.PartNumber, params.TotalParts, etag)

		// Send part to channel
		params.Channel <- models.MultipartUploadPart{
			PartNumber: int32(params.PartNumber),
			ETag:       strings.ReplaceAll(etag, "\"", ""),
		}

		break
	}
}

// Download many objects from storage to local file system.
//
// Params:
//
// - projectId: Project ID
//
// - dest: Local path where downloaded files are written to. Can be relative or absolute.
//
// - hashMap: Map of local file paths to file hashes
//
// Returns map of object keys to data.
func DownloadMany(projectSlug string, dest string, hashMap map[string]string) error {
	auth.HasToken()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()

	// Get presigned URLs
	console.Verbose("Presigning all objects...")
	bodyData := lo.Map(maps.Values(hashMap), func(hash string, _ int) models.PresignOneRequest {
		return models.PresignOneRequest{
			Method: "GET",
			Key:    hash,
		}
	})
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		return err
	}

	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/many", config.I.VCS.ServerHost, projectSlug)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	presignRes := make(map[string]models.PresignResponse)
	err = json.NewDecoder(res.Body).Decode(&presignRes)
	if err != nil {
		return err
	}

	// Download objects in parallel (limited to pool size)
	pool := workerpool.New(config.I.VCS.Storage.DownloadPoolSize)
	bar := progressbar.Default(int64(len(hashMap)))
	for key, r := range presignRes {
		// NOTE: ARGUMENTS MUST BE OUTSIDE OF SUBMITTED FUNCTION
		path := util.ReverseLookup(hashMap, key)
		if path == "" {
			return console.Error("Unknown file hash \"%s\"", key)
		}
		params := downloadParams{
			Destination: dest,
			FilePath:    path,
			URL:         r.URLs[0],
			Bar:         bar,
		}
		pool.Submit(func() {
			download(ctx, params)
		})
	}

	// Wait for downloads to finish
	pool.StopWait()

	endTime := time.Now()
	console.Info("Downloaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type downloadParams struct {
	Destination string
	FilePath    string
	URL         string
	Bar         *progressbar.ProgressBar
}

// Download object from storage to local file system.
// Intended to be called as a goroutine.
func download(ctx context.Context, params downloadParams) {
	defer params.Bar.Add(1)

	// Download object using presigned GET URL
	res, err := http.Get(params.URL)
	if err != nil {
		panic(console.Error("Failed to download file \"%s\"", params.FilePath))
	}
	defer res.Body.Close()

	// Create local file directory recursively
	path := filepath.Join(params.Destination, params.FilePath)
	dirPath := filepath.Dir(path)
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		panic(console.Error("Failed to create directory \"%s\": %v", dirPath, err))
	}

	// Create file for downloaded data, which may be compressed
	dFile, err := os.Create(path)
	if err != nil {
		panic(console.Error("Failed to create file \"%s\": %v", path, err))
	}
	defer dFile.Close()

	// Write downloaded file
	_, err = io.Copy(dFile, res.Body)
	if err != nil {
		panic(console.Error("Failed to write downloaded file \"%s\": %v", path, err))
	}

	// Read downloaded file
	dData, err := os.ReadFile(path)
	if err != nil {
		panic(console.Error("Failed to read downloaded file \"%s\": %v", path, err))
	}

	// Check for slow-down response
	// Filebase sends a "slow down" XML error response when sending requests too rapidly from a single IP
	// NOTE: Storage providers other than Filebase are not currently handled
	if len(dData) == len([]byte(constants.SlowDownFileContents)) && string(dData) == constants.SlowDownFileContents {
		// TODO: Retry file later on until successful, up to a limit
		panic(console.Error("Received slow-down error from storage provider for file \"%s\". This shouldn't happen, please contact support!", params.FilePath))
	}

	// Check if zstd compressed
	if header := hex.EncodeToString(dData[:4]); header == constants.ZstdHeader {
		console.Verbose("Decompressing file \"%s\"...", path)
		// File is compressed via zstd, decompress it
		//
		// Rename compressed file to .zst extension
		compFilePath := path + ".zst"
		err = os.Rename(path, compFilePath)
		if err != nil {
			panic(console.Error("Failed to rename compressed file \"%s\" to \"%s\": %v", path, compFilePath, err))
		}

		// Create file for decompressed data
		dcFile, err := os.Create(path)
		if err != nil {
			panic(console.Error("Failed to create file \"%s\" for decompression: %v", path, err))
		}
		defer dcFile.Close()

		// Decompress file into local file
		err = Decompress(bytes.NewBuffer(dData), dcFile)
		if err != nil {
			panic(console.Error("Failed to decompress file \"%s\": %v", path, err))
		}

		// Delete compressed file
		err = os.Remove(compFilePath)
		if err != nil {
			panic(console.Error("Failed to delete compressed file \"%s\": %v", path, err))
		}

		console.Verbose("File \"%s\" decompressed successfully", path)
	}
}
