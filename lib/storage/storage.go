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
	FileSize    int64
	ContentType string
}

// Upload many objects to storage.
//
// Params:
//
// - projectSlug: Project slug (<team_name>/<project_name>)
//
// - hashMap: Map of local file paths to file hashes (which are used as object keys)
func UploadMany(projectSlug string, hashMap map[string]string) error {
	auth.HasToken()

	console.Verbose("[UploadMany] Authenticated")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Presign objects in chunks
	// This is done in chunks to avoid Stytch rate limiting due to the sheer amount of authentication requests
	console.Verbose("Hash map: %+v", hashMap)
	console.Verbose("Chunking hash map...")
	hashMapChunked := util.ChunkMap(hashMap, config.I.VCS.Storage.PresignChunkSize)
	console.Verbose("Hash map chunked: %+v", hashMapChunked)
	presignResponses := []models.PresignResponse{}
	additionalData := make(map[string]AdditionalPresignData) // map of file path to additional data
	for chunkIdx, hashMapChunk := range hashMapChunked {
		console.Verbose("Presigning chunk %d/%d...", chunkIdx+1, len(hashMapChunked))
		bodyData := make(map[string]models.PresignOneRequestBody) // map of file path to req body data

		console.Verbose("  Building PresignMany request body...")
		for filePath, hash := range hashMapChunk {
			// Get file size
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				panic(console.Error("Failed to get file info for file \"%s\": %v", filePath, err))
			}
			fileSize := fileInfo.Size()

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

			// Get presigned URL for uploading the object later
			bodyData[filePath] = models.PresignOneRequestBody{
				Method:      "PUT",
				Key:         hash,
				ContentType: contentType,
				Multipart:   fileSize > config.I.VCS.Storage.PartSize,
				Size:        fileSize,
			}

			// Save additional data calculated above
			// This is used to prevent fetching this information again later (performance reasons)
			additionalData[filePath] = AdditionalPresignData{
				FileSize:    fileSize,
				ContentType: contentType,
			}
		}

		console.Verbose("  Request body built")
		console.Verbose("  Sending request...")

		bodyJson, err := json.Marshal(bodyData)
		if err != nil {
			panic(console.Error("Error marshalling presign request body: %v", err))
		}

		httpClient := http.Client{}
		reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/many", config.I.VCS.ServerHost, projectSlug)
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
		var newResponses []models.PresignResponse
		err = json.NewDecoder(res.Body).Decode(&newResponses)
		if err != nil {
			panic(console.Error("Error parsing presign response: %v", err))
		}

		presignResponses = append(presignResponses, newResponses...)
		console.Verbose("  Chunk presigned successfully")
	}

	startTime := time.Now()
	bar := progressbar.Default(int64(len(hashMap)))

	// Upload objects in parallel
	var wg sync.WaitGroup
	for _, presignRes := range presignResponses {
		wg.Add(1)
		hash := presignRes.Key
		path := util.ReverseLookup(hashMap, hash)
		go upload(ctx, uploadParams{
			UploadID:    presignRes.UploadID,
			URLs:        presignRes.URLs,
			ProjectSlug: projectSlug,
			FilePath:    path,
			ContentType: additionalData[path].ContentType,
			Size:        additionalData[path].FileSize,
			Hash:        hash,
			Bar:         bar,
			WG:          &wg,
		})
	}

	// Wait for all uploads to finish
	wg.Wait()

	endTime := time.Now()
	console.Verbose("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type uploadParams struct {
	UploadID    string
	URLs        []string
	ProjectSlug string
	FilePath    string
	ContentType string
	Size        int64
	Hash        string
	Bar         *progressbar.ProgressBar
	WG          *sync.WaitGroup
}

// Upload object to storage. Can be multipart or in full.
// Intended to be called as a goroutine.
func upload(ctx context.Context, params uploadParams) {
	defer params.WG.Done()
	defer params.Bar.Add(1)

	if params.Size < config.I.VCS.Storage.PartSize {
		// NON-MULTIPART
		//
		// Read file into byte array
		fileBytes, err := os.ReadFile(params.FilePath)
		if err != nil {
			panic(console.Error("Failed to read file \"%s\": %v", params.FilePath, err))
		}

		// Upload file
		console.Verbose("[%s] Uploading in full...", params.Hash)
		uploadSingle(ctx, params, params.ContentType, params.Size, fileBytes)
	} else {
		// MULTIPART
		//
		// Open file
		file, err := os.Open(params.FilePath)
		if err != nil {
			panic(console.Error("Failed to open file \"%s\": %v", params.FilePath, err))
		}

		// Compress file
		tempFile, err := os.CreateTemp(filepath.Dir(params.FilePath), filepath.Base(params.FilePath)+".tmp-")
		if err != nil {
			panic(console.Error("Failed to open temp file for compression: %v", err))
		}
		tempFilePath := tempFile.Name()
		console.Verbose("[%s] Compressing; temp file: \"%s\"...", params.Hash, tempFilePath)
		err = Compress(file, tempFile)
		if err != nil {
			panic(console.Error("Failed to compress file \"%s\": %v", params.FilePath, err))
		}
		defer tempFile.Close()

		// Read compressed file into byte array
		fileBytes, err := os.ReadFile(tempFilePath)
		if err != nil {
			panic(console.Error("Failed to read file \"%s\" after compression: %v", params.FilePath, err))
		}
		fileSize := int64(len(fileBytes))

		// Delete compressed file
		err = os.Remove(tempFilePath)
		if err != nil {
			panic(console.Error("Failed to delete compressed file \"%s\": %v", tempFilePath, err))
		}

		// Upload multipart file
		console.Verbose("[%s] Uploading in chunks...", params.Hash)
		uploadMultipart(ctx, params, params.ContentType, fileSize, fileBytes)
	}
}

// Upload object in full to storage.
// Intended to be called as a goroutine.
func uploadSingle(ctx context.Context, params uploadParams, contentType string, fileSize int64, fileBytes []byte) {
	// Wait until rate limiter frees up before uploading to storage
	err := config.I.RateLimiter.Wait(ctx)
	if err != nil {
		panic(err)
	}

	// Presign object
	bodyData := models.PresignOneRequestBody{
		Key:         params.Hash,
		Multipart:   false,
		Size:        fileSize,
		ContentType: contentType,
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		panic(console.Error("Error marshalling presign request body while presigning upload for file \"%s\": %v", params.FilePath, err))
	}

	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/put", config.I.VCS.ServerHost, params.ProjectSlug)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		panic(console.Error("Error presigning file \"%s\": %v", params.FilePath, err))
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(console.Error("Error presigning file \"%s\": %v", params.FilePath, err))
	}
	defer res.Body.Close()

	// Parse response
	var presignRes models.PresignResponse
	err = json.NewDecoder(res.Body).Decode(&presignRes)
	if err != nil {
		panic(console.Error("Error parsing presign response for file \"%s\": %v", params.FilePath, err))
	}

	if presignRes.UploadID != "" {
		panic(console.Error("Presigned upload returned with an upload ID for non-multipart upload of file \"%s\"", params.FilePath))
	}

	if len(presignRes.URLs) != 1 {
		panic(console.Error("Presigned upload returned with %d URLs for non-multipart upload of file \"%s\"", len(presignRes.URLs), params.FilePath))
	}

	// Upload using presigned URL
	console.Verbose("[%s] Uploading...", params.Hash)
	url := presignRes.URLs[0]
	req, err = http.NewRequest("PUT", url, bytes.NewBuffer(fileBytes))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", contentType)
	res, err = httpClient.Do(req)
	if err != nil {
		panic(console.Error("Error uploading file \"%s\": %v", params.FilePath, err))
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(console.Error("Error uploading file \"%s\": %v", params.FilePath, err))
	}
	res.Body.Close()
	console.Verbose("[%s] Uploaded", params.Hash)
}

// Upload a file in chunks to storage.
// Intended to be called as a goroutine.
func uploadMultipart(ctx context.Context, params uploadParams, contentType string, fileSize int64, fileBytes []byte) {
	// Wait until rate limiter frees up before uploading to storage
	err := config.I.RateLimiter.Wait(ctx)
	if err != nil {
		panic(err)
	}

	// Presign object
	console.Verbose("[%s] Presigning...", params.Hash)
	bodyData := models.PresignOneRequestBody{
		Key:         params.Hash,
		Multipart:   true,
		Size:        fileSize,
		ContentType: contentType,
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		panic(console.Error("Error marshalling presign request body while presigning upload for file \"%s\": %v", params.FilePath, err))
	}

	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/put", config.I.VCS.ServerHost, params.ProjectSlug)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		panic(console.Error("Error presigning file \"%s\": %v", params.FilePath, err))
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(console.Error("Error presigning file \"%s\": %v", params.FilePath, err))
	}
	defer res.Body.Close()

	// Parse response
	var presignRes models.PresignResponse
	err = json.NewDecoder(res.Body).Decode(&presignRes)
	if err != nil {
		panic(console.Error("Error parsing presign response for file \"%s\": %v", params.FilePath, err))
	}

	if presignRes.UploadID == "" {
		panic(console.Error("Presigned multipart upload returned with no upload ID for file \"%s\"", params.FilePath))
	}

	if len(presignRes.URLs) == 0 {
		panic(console.Error("No URLs returned while presigning multipart upload for file \"%s\"", params.FilePath))
	}

	// Split file into chunks
	chunks := [][]byte{}
	var start int64
	remaining := fileSize
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
	for i, url := range presignRes.URLs {
		wg.Add(1)
		partNum := i + 1
		go uploadPart(ctx, uploadPartParams{
			ProjectID:   params.ProjectSlug,
			URL:         url,
			Hash:        params.Hash,
			ContentType: contentType,
			PartNumber:  partNum,
			PartData:    chunks[i],
			TotalParts:  totalParts,
			WG:          &wg,
			Channel:     ch,
		})
		if err != nil {
			panic(console.Error("Error uploading part %d of file \"%s\": %v", partNum, params.FilePath, err))
		}
		p := <-ch
		parts = append(parts, p)
	}

	wg.Wait()

	// Complete multipart upload
	console.Verbose("[%s] Completing...", params.Hash)
	complBodyData := models.CompleteMultipartUploadRequestBody{
		UploadId: presignRes.UploadID,
		Key:      params.Hash,
		Parts:    parts,
	}
	complBodyJson, err := json.Marshal(complBodyData)
	if err != nil {
		panic(console.Error("Error marshalling \"complete multipart upload\" request body for file \"%s\": %v", params.FilePath, err))
	}

	reqUrl = fmt.Sprintf("%s/projects/%s/storage/multipart/complete", config.I.VCS.ServerHost, params.ProjectSlug)
	req, err = http.NewRequest("POST", reqUrl, bytes.NewBuffer(complBodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err = httpClient.Do(req)
	if err != nil {
		panic(console.Error("Error completing multipart upload for file \"%s\": %v", params.FilePath, err))
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(console.Error("Error completing multipart upload for file \"%s\": %v", params.FilePath, err))
	}
	res.Body.Close()
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

	// Wait until rate limiter frees up before uploading to storage
	err := config.I.RateLimiter.Wait(ctx)
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
		panic(console.Error("[%s] Error uploading part %d: %v", params.Hash, params.PartNumber, err))
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		panic(console.Error("[%s] Error uploading part %d: %v", params.Hash, params.PartNumber, err))
	}
	defer res.Body.Close()

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
	bodyData := lo.Map(maps.Values(hashMap), func(hash string, _ int) models.PresignOneRequestBody {
		return models.PresignOneRequestBody{
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
	var presignResponses []models.PresignResponse
	err = json.NewDecoder(res.Body).Decode(&presignResponses)
	if err != nil {
		return err
	}

	// Download objects in parallel (limited to pool size)
	pool := workerpool.New(config.I.VCS.Storage.DownloadPoolSize)
	bar := progressbar.Default(int64(len(hashMap)))
	for _, r := range presignResponses {
		// NOTE: ARGUMENTS MUST BE OUTSIDE OF SUBMITTED FUNCTION
		path := util.ReverseLookup(hashMap, r.Key)
		if path == "" {
			return console.Error("Unknown file hash \"%s\"", r.Key)
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
