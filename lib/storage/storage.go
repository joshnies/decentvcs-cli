package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/util"
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
// - hashMap: Map of local file paths to file hashes
//
func UploadMany(projectId string, hashMap map[string]string) error {
	gc := auth.Validate()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()

	// Chunk uploads
	// TODO: Use limit-based approach where chunks are uploaded in parallel up to a limit, where they wait in a queue
	// until the current upload count goes below the limit again.
	chunks := util.ChunkMap(hashMap, 32)

	// For each chunk...
	for _, chunk := range chunks {
		// Get presigned URLs
		bodyData := map[string][]string{
			"keys": maps.Values(chunk),
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

		// Setup wait group
		var wg sync.WaitGroup
		wg.Add(len(chunk))

		// Setup multi-progress bar container
		p := mpb.New()

		// Upload objects in parallel
		for hash, url := range hashUrlMap {
			path := util.ReverseLookup(hashMap, hash)
			go uploadRoutine(ctx, uploadRoutineParams{
				FilePath: path,
				URL:      url,
				WG:       &wg,
				Progress: p,
			})
		}

		// Wait for uploads to finish
		wg.Wait()
		p.Wait()
	}

	endTime := time.Now()
	console.Info("Uploaded %d files in %s", len(hashMap), endTime.Sub(startTime))

	return nil
}

type uploadRoutineParams struct {
	FilePath string
	URL      string
	WG       *sync.WaitGroup
	Progress *mpb.Progress
}

func uploadRoutine(ctx context.Context, params uploadRoutineParams) {
	defer params.WG.Done()

	// Read local file
	file, err := os.Open(params.FilePath)
	if err != nil {
		console.ErrorPrintV("Failed to open file \"%s\": %v", params.FilePath, err)
		panic(err)
	}

	// Get local file size
	fileInfo, err := file.Stat()
	if err != nil {
		console.ErrorPrint("Failed to stat file \"%s\": %v", params.FilePath, err)
		panic(err)
	}

	total := fileInfo.Size()

	// Add progress bar
	barName := filepath.Base(params.FilePath)

	if len(barName) > 20 {
		barName = barName[:17] + "..."
	}

	bar := params.Progress.AddBar(total,
		mpb.PrependDecorators(
			decor.Name(barName, decor.WC{W: 20, C: decor.DidentRight}),
		),
		mpb.AppendDecorators(
			decor.CountersKibiByte("% .2f / % .2f ", decor.WCSyncSpace),
		),
	)
	proxyReader := bar.ProxyReader(file)
	defer proxyReader.Close()

	// Get MIME type
	var contentType string
	mtype, err := mimetype.DetectReader(proxyReader)
	if err != nil {
		console.Warning("Failed to detect MIME type for file \"%s\", using default", params.FilePath)
		contentType = "application/octet-stream"
	}

	contentType = mtype.String()

	// Upload object using presigned PUT URL
	_, err = httpw.Put(httpw.RequestParams{
		URL:         params.URL,
		Body:        proxyReader,
		ContentType: contentType,
	})
	if err != nil {
		console.ErrorPrint("Failed to upload file \"%s\": %v", params.FilePath, err)
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
		console.ErrorPrint("Failed to download file \"%s\": %v", params.FilePath, err)
		panic(err)
	}
	defer res.Body.Close()

	// Write to local file
	path := filepath.Join(params.ProjectPath, params.FilePath)
	file, err := os.Create(path)
	if err != nil {
		console.ErrorPrint("Failed to create file \"%s\": %v", path, err)
		panic(err)
	}

	// Copy response body to local file
	_, err = io.Copy(file, res.Body)
	if err != nil {
		console.ErrorPrint("Failed to write file \"%s\": %v", path, err)
		panic(err)
	}
}
