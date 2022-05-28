package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/util"
	"github.com/joshnies/qc/models"
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
		params := uploadRoutineParams{
			ProjectID: projectId,
			FilePath:  path,
			Hash:      hash,
			Bar:       bar,
			GC:        &gc,
		}
		pool.Submit(func() {
			uploadRoutine(ctx, params)
		})
	}
	pool.StopWait()

	endTime := time.Now()
	console.Verbose("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type uploadRoutineParams struct {
	ProjectID string
	FilePath  string
	Hash      string
	Bar       *progressbar.ProgressBar
	GC        *models.GlobalConfig
}

// Upload object to storage. Can be multipart or in full.
func uploadRoutine(ctx context.Context, params uploadRoutineParams) {
	defer params.Bar.Add(1)

	// Open local file
	file, err := os.Open(params.FilePath)
	if err != nil {
		panic(console.Error("Failed to open file \"%s\": %v", params.FilePath, err))
	}
	defer file.Close()

	// Get file size
	info, err := file.Stat()
	if err != nil {
		panic(console.Error("Failed to get file info for file \"%s\": %v", params.FilePath, err))
	}
	fileSize := info.Size()

	// Read file into byte array
	fileBytes := make([]byte, fileSize)
	_, err = file.Read(fileBytes)
	if err != nil {
		panic(console.Error("Failed to read file \"%s\": %v", params.FilePath, err))
	}

	// Get MIME type
	// var contentType string
	// mtype, err := mimetype.DetectReader(file)
	// if err != nil {
	// 	contentType = "application/octet-stream"
	// 	console.Warning("Failed to detect MIME type for file \"%s\", using default \"%s\"", params.FilePath, contentType)
	// } else {
	// 	contentType = mtype.String()
	// }

	contentType := "application/octet-stream"

	if fileSize < 5*1024*1024 {
		console.Verbose("[%s] Uploading in full...", params.Hash)
		uploadRoutineSingle(ctx, params, contentType, fileSize, fileBytes)
	} else {
		console.Verbose("[%s] Uploading in chunks...", params.Hash)
		uploadRoutineMultipart(ctx, params, contentType, fileSize, fileBytes)
	}
}

// Upload object in full to storage.
func uploadRoutineSingle(ctx context.Context, params uploadRoutineParams, contentType string, fileSize int64, fileBytes []byte) {
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

	res, err := httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s/storage/presign/put", params.ProjectID),
		Body:        bytes.NewBuffer(bodyJson),
		AccessToken: params.GC.Auth.AccessToken,
	})
	if err != nil {
		panic(console.Error("Error presigning file \"%s\": %v", params.FilePath, err))
	}

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
	url := presignRes.URLs[0]
	console.Verbose("[%s] Uploading...", params.Hash)
	_, err = httpw.Put(httpw.RequestParams{
		URL:         url,
		Body:        bytes.NewBuffer(fileBytes),
		ContentType: contentType,
	})
	if err != nil {
		panic(console.Error("Error uploading file \"%s\": %v", params.FilePath, err))
	}

	console.Verbose("[%s] Uploaded", params.Hash)
}

// Upload a file in chunks to storage.
func uploadRoutineMultipart(ctx context.Context, params uploadRoutineParams, contentType string, fileSize int64, fileBytes []byte) {
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

	res, err := httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s/storage/presign/put", params.ProjectID),
		Body:        bytes.NewBuffer(bodyJson),
		AccessToken: params.GC.Auth.AccessToken,
	})
	if err != nil {
		panic(console.Error("Error presigning file \"%s\": %v", params.FilePath, err))
	}

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

	// Upload object using presigned PUT URLs
	parts := []models.MultipartUploadPart{}
	var start, current int64
	remaining := fileSize
	for i, url := range presignRes.URLs {
		console.Verbose("[%s] (Part %d/%d) Uploading...", params.Hash, i+1, len(presignRes.URLs))

		// Get file part as byte array
		if remaining < config.I.Storage.PartSize {
			current = remaining
		} else {
			current = config.I.Storage.PartSize
		}
		partBytes := fileBytes[start : start+current]

		res, err = httpw.Put(httpw.RequestParams{
			URL:         url,
			Body:        bytes.NewReader(partBytes),
			ContentType: contentType,
		})
		if err != nil {
			panic(console.Error("Error uploading part %d of file \"%s\": %v", i, params.FilePath, err))
		}

		// Validate response headers
		etag := res.Header.Get("etag")
		if etag == "" {
			panic(console.Error("No \"etag\" header returned for part %d of file \"%s\"", i, params.FilePath))
		}

		parts = append(parts, models.MultipartUploadPart{
			PartNumber: int32(i + 1),
			ETag:       etag,
		})

		// Update loop variables
		partBytesLen := int64(len(partBytes))
		start += partBytesLen
		remaining -= partBytesLen

		console.Verbose("[%s] (Part %d/%d) Uploaded", params.Hash, i+1, len(presignRes.URLs))
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
	_, err = httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s/storage/multipart/complete", params.ProjectID),
		Body:        bytes.NewBuffer(complBodyJson),
		AccessToken: params.GC.Auth.AccessToken,
	})
	if err != nil {
		panic(console.Error("Error completing multipart upload for file \"%s\": %v", params.FilePath, err))
	}

	console.Verbose("[%s] Complete", params.Hash)
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

	res, err := httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s/storage/presign/many", projectId),
		Body:        bytes.NewBuffer(bodyJson),
		AccessToken: gc.Auth.AccessToken,
	})
	if err != nil {
		return err
	}

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
		path := util.ReverseLookup(hashMap, hash)
		params := downloadRoutineParams{
			ProjectPath: projectPath,
			FilePath:    path,
			URL:         url,
			Bar:         bar,
		}
		pool.Submit(func() {
			downloadRoutine(ctx, params)
		})
	}

	// Wait for downloads to finish
	pool.StopWait()

	endTime := time.Now()
	console.Info("Downloaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type downloadRoutineParams struct {
	ProjectPath string
	FilePath    string
	URL         string
	Bar         *progressbar.ProgressBar
}

func downloadRoutine(ctx context.Context, params downloadRoutineParams) {
	defer params.Bar.Add(1)

	// Download object using presigned GET URL
	res, err := httpw.Get(params.URL, "")
	if err != nil {
		console.ErrorPrint("Failed to download file \"%s\"", params.FilePath)
		panic(err)
	}
	defer res.Body.Close()

	// Create local file directory recursively
	path := filepath.Join(params.ProjectPath, params.FilePath)
	dirPath := filepath.Dir(path)
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		console.ErrorPrint("Failed to create directory \"%s\": %v", dirPath, err)
		panic(err)
	}

	// Create file (overwrite)
	file, err := os.Create(path)
	if err != nil {
		console.ErrorPrint("Failed to create file \"%s\": %v", path, err)
		panic(err)
	}
	defer file.Close()

	// Copy response body to local file
	_, err = io.Copy(file, res.Body)
	if err != nil {
		console.ErrorPrint("Failed to write file \"%s\": %v", path, err)
		panic(err)
	}
}
