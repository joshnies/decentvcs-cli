package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/lib/util"
	"github.com/joshnies/quanta/models"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/maps"
)

// Upload many objects to storage.
//
// Params:
//
// - projectId: Project ID
//
// - hashMap: Map of local file paths to file hashes (which are used as object keys)
//
func UploadMany(projectId string, hashMap map[string]string) error {
	gc := auth.Validate()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()
	bar := progressbar.Default(int64(len(hashMap)))

	// Upload objects in parallel
	pool := workerpool.New(config.I.Storage.UploadPoolSize)
	for path, hash := range hashMap {
		// NOTE: ARGUMENTS MUST BE OUTSIDE OF SUBMITTED FUNCTION
		params := uploadParams{
			ProjectID: projectId,
			FilePath:  path,
			Hash:      hash,
			Bar:       bar,
			GC:        &gc,
		}
		pool.Submit(func() {
			upload(ctx, params)
		})
	}

	// Wait for uploads to finish
	pool.StopWait()

	endTime := time.Now()
	console.Verbose("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type uploadParams struct {
	ProjectID string
	FilePath  string
	Hash      string
	Bar       *progressbar.ProgressBar
	GC        *models.GlobalConfig
}

// Upload object to storage. Can be multipart or in full.
// Intended to be called as a goroutine.
func upload(ctx context.Context, params uploadParams) {
	defer params.Bar.Add(1)

	// Open local file
	file, err := os.Open(params.FilePath)
	if err != nil {
		panic(console.Error("Failed to open file \"%s\": %v", params.FilePath, err))
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		panic(console.Error("Failed to get file info for file \"%s\": %v", params.FilePath, err))
	}
	fileSize := fileInfo.Size()

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

	if fileSize < config.I.Storage.PartSize {
		// NON-MULTIPART
		//
		// Read file into byte array
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(console.Error("Failed to read file \"%s\": %v", params.FilePath, err))
		}

		// Upload file
		console.Verbose("[%s] Uploading in full...", params.Hash)
		uploadSingle(ctx, params, contentType, fileSize, fileBytes)
	} else {
		// MULTIPART
		//
		// Compress file
		tempFile, err := os.CreateTemp(filepath.Dir(params.FilePath), filepath.Base(params.FilePath)+".temp-")
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
		fileBytes, err := ioutil.ReadFile(tempFilePath)
		if err != nil {
			panic(console.Error("Failed to read file \"%s\" after compression: %v", params.FilePath, err))
		}
		fileSize = int64(len(fileBytes))

		// Delete compressed file
		err = os.Remove(tempFilePath)
		if err != nil {
			panic(console.Error("Failed to delete compressed file \"%s\": %v", tempFilePath, err))
		}

		// Upload multipart file
		console.Verbose("[%s] Uploading in chunks...", params.Hash)
		uploadMultipart(ctx, params, contentType, fileSize, fileBytes)
	}
}

// Upload object in full to storage.
// Intended to be called as a goroutine.
func uploadSingle(ctx context.Context, params uploadParams, contentType string, fileSize int64, fileBytes []byte) {
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
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/put", config.I.API.Host, params.ProjectID)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", params.GC.Auth.AccessToken))
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
	var presignRes models.PresignOneResponse
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
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/put", config.I.API.Host, params.ProjectID)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", params.GC.Auth.AccessToken))
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
	var presignRes models.PresignOneResponse
	err = json.NewDecoder(res.Body).Decode(&presignRes)
	if err != nil {
		panic(console.Error("Error parsing presign response for file \"%s\": %v", params.FilePath, err))
	}

	if presignRes.UploadID == "" {
		panic(console.Error("Presigned multipart upload returned with no upload ID for file \"%s\"", params.FilePath))
	}

	if len(presignRes.URLs) <= 1 {
		panic(console.Error("Presigned multipart upload returned with %d URLs for file \"%s\"", len(presignRes.URLs), params.FilePath))
	}

	// Split file into chunks
	chunks := [][]byte{}
	var start int64
	remaining := fileSize
	for remaining > 0 {
		chunkSize := int64(math.Min(float64(remaining), float64(config.I.Storage.PartSize)))
		chunks = append(chunks, fileBytes[start:start+chunkSize])
		start += chunkSize
		remaining -= chunkSize
	}

	// Upload parts in parallel (limited to pool size)
	// TODO: Parallelize this
	parts := []models.MultipartUploadPart{}
	totalParts := len(chunks)
	for i, url := range presignRes.URLs {
		partNum := i + 1
		p, err := uploadPart(ctx, uploadPartParams{
			ProjectID:   params.ProjectID,
			URL:         url,
			Hash:        params.Hash,
			ContentType: contentType,
			PartNumber:  partNum,
			PartData:    chunks[i],
			TotalParts:  totalParts,
		})
		if err != nil {
			panic(console.Error("Error uploading part %d of file \"%s\": %v", partNum, params.FilePath, err))
		}
		parts = append(parts, p)
	}

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

	reqUrl = fmt.Sprintf("%s/projects/%s/storage/multipart/complete", config.I.API.Host, params.ProjectID)
	req, err = http.NewRequest("POST", reqUrl, bytes.NewBuffer(complBodyJson))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", params.GC.Auth.AccessToken))
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
}

// Upload part to storage for a multipart upload.
// Must be called as a goroutine.
func uploadPart(ctx context.Context, params uploadPartParams) (models.MultipartUploadPart, error) {
	console.Verbose("[%s] (Part %d/%d) Uploading...", params.Hash, params.PartNumber, params.TotalParts)

	// Upload part
	httpClient := http.Client{}
	req, err := http.NewRequest("PUT", params.URL, bytes.NewReader(params.PartData))
	if err != nil {
		return models.MultipartUploadPart{}, err
	}
	req.Header.Add("Content-Type", params.ContentType)
	res, err := httpClient.Do(req)
	if err != nil {
		return models.MultipartUploadPart{}, console.Error("[%s] Error uploading part %d: %v", params.Hash, params.PartNumber, err)
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return models.MultipartUploadPart{}, console.Error("[%s] Error uploading part %d: %v", params.Hash, params.PartNumber, err)
	}
	defer res.Body.Close()

	// Validate response headers
	etag := res.Header.Get("etag")
	if etag == "" {
		return models.MultipartUploadPart{}, console.Error("[%s] No \"etag\" header returned for part %d", params.Hash, params.PartNumber)
	}
	console.Verbose("[%s] (Part %d/%d) Uploaded; ETag: %s", params.Hash, params.PartNumber, params.TotalParts, etag)

	// Send part to channel
	return models.MultipartUploadPart{
		PartNumber: int32(params.PartNumber),
		ETag:       strings.ReplaceAll(etag, "\"", ""),
	}, nil
}

// Download many objects from storage to local file system.
//
// Params:
//
// - projectId: Project ID
//
// - projectPath: Local file path to project path. Can be relative or absolute.
//
// - hashMap: Map of local file paths to file hashes
//
// Returns map of object keys to data.
//
func DownloadMany(projectId string, projectPath string, hashMap map[string]string) error {
	gc := auth.Validate()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()

	// Get presigned URLs
	console.Verbose("Presigning all objects...")
	bodyData := models.PresignManyRequestBody{
		Keys: maps.Values(hashMap),
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		return err
	}

	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/storage/presign/many", config.I.API.Host, projectId)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
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
	var hashUrlMap map[string]string
	err = json.NewDecoder(res.Body).Decode(&hashUrlMap)
	if err != nil {
		return err
	}

	// Download objects in parallel (limited to pool size)
	pool := workerpool.New(config.I.Storage.DownloadPoolSize)
	bar := progressbar.Default(int64(len(hashMap)))
	for hash, url := range hashUrlMap {
		// NOTE: ARGUMENTS MUST BE OUTSIDE OF SUBMITTED FUNCTION
		path := util.ReverseLookup(hashMap, hash)
		if path == "" {
			return console.Error("Unknown file hash \"%s\"", hash)
		}
		params := downloadParams{
			ProjectPath: projectPath,
			FilePath:    path,
			URL:         url,
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
	ProjectPath string
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
	path := filepath.Join(params.ProjectPath, params.FilePath)
	dirPath := filepath.Dir(path)
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		panic(console.Error("Failed to create directory \"%s\": %v", dirPath, err))
	}

	// Create file (overwrite)
	file, err := os.Create(path)
	if err != nil {
		panic(console.Error("Failed to create file \"%s\": %v", path, err))
	}
	defer file.Close()

	// Decompress file into local file
	err = Decompress(res.Body, file)
	if err != nil {
		panic(console.Error("Failed to decompress and write file \"%s\": %v", path, err))
	}
}
