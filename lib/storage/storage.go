package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gammazero/workerpool"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/util"
	"github.com/schollz/progressbar/v3"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
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
	console.Verbose("Presigning all objects...", len(hashMap))
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
	pool := workerpool.New(128)
	bar := progressbar.Default(int64(len(hashMap)))

	// Upload objects in parallel (limited to pool size)
	for hash, url := range hashUrlMap {
		path := util.ReverseLookup(hashMap, hash)
		pool.Submit(func() {
			uploadRoutine(ctx, uploadRoutineParams{
				FilePath: path,
				URL:      url,
				Bar:      bar,
			})
		})
	}

	// Wait for uploads to finish
	pool.StopWait()

	endTime := time.Now()
	console.Verbose("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))
	return nil
}

type uploadRoutineParams struct {
	FilePath string
	URL      string
	WG       *sync.WaitGroup
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
	res, err := httpw.Put(httpw.RequestParams{
		URL:         params.URL,
		Body:        file,
		ContentType: contentType,
	})
	if err != nil {
		console.ErrorPrint("Failed to upload file \"%s\"", params.FilePath)

		// Print response dump
		if res != nil {
			dump, err := httputil.DumpResponse(res, true)
			if err != nil {
				console.ErrorPrint("(failed to dump response); %v", err)
			} else {
				console.ErrorPrint("Response:\n%s", string(dump))
			}
		} else {
			console.ErrorPrint("(no response)")
		}

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

	// Chunk downloads
	// TODO: Use limit-based approach where chunks are downloaded in parallel up to a limit, where they wait in a queue
	// until the current download count goes below the limit again.
	chunks := util.ChunkMap(hashMap, 32)

	// For each chunk...
	for i, chunk := range chunks {
		// Get presigned URLs
		bodyData := map[string][]string{
			"keys": maps.Values(chunk),
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

		// Setup wait group
		var wg sync.WaitGroup
		wg.Add(len(chunk))

		// Setup progress bar
		p := mpb.New()
		bar := p.AddBar(int64(len(chunk)),
			mpb.PrependDecorators(
				decor.Name(fmt.Sprintf("Downloading chunk %d/%d", i+1, len(chunks)), decor.WC{W: 20, C: decor.DidentRight}),
			),
			mpb.AppendDecorators(
				decor.CountersNoUnit("%d / %d ", decor.WCSyncSpace),
			),
		)

		// Download objects in parallel
		for hash, url := range hashUrlMap {
			path := util.ReverseLookup(hashMap, hash)
			go downloadRoutine(ctx, &downloadRoutineParams{
				ProjectPath: projectPath,
				FilePath:    path,
				URL:         url,
				WG:          &wg,
				ProgressBar: bar,
			})
		}

		// Wait for downloads to finish
		wg.Wait()
		p.Wait()
	}

	endTime := time.Now()
	console.Info("Downloaded %d files in %s", len(hashMap), endTime.Sub(startTime))

	return nil
}

type downloadRoutineParams struct {
	ProjectPath string
	FilePath    string
	URL         string
	WG          *sync.WaitGroup
	ProgressBar *mpb.Bar
}

func downloadRoutine(ctx context.Context, params *downloadRoutineParams) {
	defer params.WG.Done()
	defer params.ProgressBar.Increment()

	// Download object using presigned GET URL
	res, err := httpw.Get(params.URL, "")
	if err != nil {
		console.ErrorPrint("Failed to download file \"%s\"", params.FilePath)

		// Print response dump
		if res != nil {
			dump, err := httputil.DumpResponse(res, true)
			if err != nil {
				console.ErrorPrint("(failed to dump response); %v", err)
			} else {
				console.ErrorPrint("Response:\n%s", string(dump))
			}
		} else {
			console.ErrorPrint("(no response)")
		}

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
