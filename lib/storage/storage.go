package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/util"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()

	// Initialize S3 client
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           "https://s3.filebase.com",
				SigningRegion: "us-east-1",
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awscfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithEndpointResolverWithOptions(customResolver))
	if err != nil {
		console.ErrorPrintV("Failed to load AWS SDK config: %v", err)
		return console.Error(constants.ErrMsgInternal)
	}

	client := s3.NewFromConfig(awscfg)

	// Chunk uploads
	chunks := util.ChunkMap(hashMap, 256)

	// For each chunk...
	for _, chunk := range chunks {
		// Setup wait group
		var wg sync.WaitGroup
		wg.Add(len(chunk))

		// Setup multi-progress bar container
		p := mpb.New()

		// Upload objects in parallel
		for path, hash := range chunk {
			go uploadRoutine(ctx, uploadRoutineParams{
				S3Client:  client,
				ProjectID: projectId,
				FilePath:  path,
				FileHash:  hash,
				WG:        &wg,
				Progress:  p,
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
	S3Client  *s3.Client
	ProjectID string
	FilePath  string
	FileHash  string
	WG        *sync.WaitGroup
	Progress  *mpb.Progress
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

	// Upload new object to S3
	key := fmt.Sprintf("%s/%s", params.ProjectID, params.FileHash)
	_, err = params.S3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &config.I.Storage.Bucket,
		Key:    &key,
		Body:   proxyReader,
	})

	if err != nil {
		console.ErrorPrintV("Failed to upload file \"%s\" with key \"%s\": %+v", params.FilePath, params.FileHash, err)
		panic(err)
	}
}
