package storage

import (
	"bytes"
	"context"
	"encoding/json"
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
	chunks := util.ChunkMap(hashMap, 256)

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

		res, err := httpw.Post(httpw.RequestInput{
			URL:         api.BuildURLf("projects/%s/presign", projectId),
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

	// Upload object using presigned URL
	_, err = httpw.Put(httpw.RequestInput{
		URL:         params.URL,
		Body:        proxyReader,
		ContentType: contentType,
	})
	if err != nil {
		console.ErrorPrint("Failed to upload file \"%s\": %v", params.FilePath, err)
		panic(err)
	}
}

// Download many objects from storage.
//
// Params:
//
// - projectId: Project ID
//
// - keys: Keys for objects to download
//
// Returns map of object keys to data.
//
func DownloadMany(projectId string, keys []string) (map[string][]byte, error) {
	// TODO: Implement
	return nil, nil
}
