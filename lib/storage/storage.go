package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gammazero/workerpool"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/util"
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

	// Get presigned URLs
	console.Verbose("Presigning all objects...")
	bodyData := map[string][]string{
		"keys": maps.Values(hashMap),
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		return err
	}

	res, err := httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s/presign/put", projectId),
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

	// TODO: Get pool size from global config
	// pool := workerpool.New(128)
	bar := progressbar.Default(int64(len(hashMap)))

	// // Upload objects in parallel (limited to pool size)
	// for hash, url := range hashUrlMap {
	// 	path := util.ReverseLookup(hashMap, hash)
	// 	pool.Submit(func() {
	// 		uploadRoutine(ctx, uploadRoutineParams{
	// 			FilePath: path,
	// 			URL:      url,
	// 			Bar:      bar,
	// 		})
	// 	})
	// }

	// // Wait for uploads to finish
	// pool.StopWait()

	// Upload objects sequentially
	for hash, url := range hashUrlMap {
		path := util.ReverseLookup(hashMap, hash)
		uploadRoutine(ctx, uploadRoutineParams{
			FilePath: path,
			URL:      url,
			Bar:      bar,
		})
	}

	endTime := time.Now()
	console.Verbose("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type uploadRoutineParams struct {
	FilePath string
	URL      string
	Bar      *progressbar.ProgressBar
}

func uploadRoutine(ctx context.Context, params uploadRoutineParams) {
	defer params.Bar.Add(1)

	// Read local file
	file, err := os.Open(params.FilePath)
	if err != nil {
		console.ErrorPrint("Failed to open file \"%s\": %v", params.FilePath, err)
		panic(err)
	}
	defer file.Close()

	// Get MIME type
	var contentType string
	mtype, err := mimetype.DetectReader(file)
	if err != nil {
		contentType = "application/octet-stream"
		console.Warning("Failed to detect MIME type for file \"%s\", using default \"%s\"", params.FilePath, contentType)
	} else {
		contentType = mtype.String()
	}

	// Upload object using presigned PUT URL
	_, err = httpw.Put(httpw.RequestParams{
		URL:         params.URL,
		Body:        file,
		ContentType: contentType,
	})
	if err != nil {
		console.ErrorPrint("Failed to upload file \"%s\"", params.FilePath)
		panic(err)
	}
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
	bodyData := map[string][]string{
		"keys": maps.Values(hashMap),
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		return err
	}

	res, err := httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s/presign/get", projectId),
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

	// TODO: Get pool size from global config
	pool := workerpool.New(128)
	bar := progressbar.Default(int64(len(hashMap)))

	// Download objects in parallel (limited to pool size)
	for hash, url := range hashUrlMap {
		path := util.ReverseLookup(hashMap, hash)
		pool.Submit(func() {
			downloadRoutine(ctx, &downloadRoutineParams{
				ProjectPath: projectPath,
				FilePath:    path,
				URL:         url,
				Bar:         bar,
			})
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

func downloadRoutine(ctx context.Context, params *downloadRoutineParams) {
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
