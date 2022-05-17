package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	chunks := util.ChunkMap(hashMap, 32)

	// For each chunk...
	for _, chunk := range chunks {
		// Setup for parallel uploads
		var wg sync.WaitGroup
		wg.Add(len(maps.Keys(chunk)))

		p := mpb.New(mpb.WithWidth(60))

		// Upload objects in parallel
		for path, hash := range chunk {
			go uploadRoutine(ctx, client, projectId, path, hash, &wg, p)
		}

		// Wait for uploads to finish
		wg.Wait()
		p.Wait()
	}

	return nil
}

func uploadRoutine(ctx context.Context, client *s3.Client, projectId string, path string, hash string, wg *sync.WaitGroup, p *mpb.Progress) {
	defer wg.Done()

	// Read local file
	file, err := os.Open(path)
	if err != nil {
		console.ErrorPrintV("Failed to open file \"%s\": %v", path, err)
		return
	}

	// Get local file size
	fileInfo, err := file.Stat()
	if err != nil {
		console.ErrorPrint("Failed to stat file \"%s\": %v", path, err)
		panic(err)
	}

	total := fileInfo.Size()

	// Add progress bar
	barName := filepath.Base(path)

	if len(barName) > 20 {
		barName = barName[:17] + "..."
	}

	bar := p.AddBar(total,
		mpb.PrependDecorators(
			decor.Name(barName, decor.WC{W: 20, C: decor.DidentRight}),
		),
		mpb.AppendDecorators(
			decor.CountersKibiByte("% .2f / % .2f "),
		),
	)
	proxyReader := bar.ProxyReader(file)
	defer proxyReader.Close()

	// Upload new object to S3
	key := fmt.Sprintf("%s/%s", projectId, hash)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &config.I.Storage.Bucket,
		Key:    &key,
		Body:   proxyReader,
	})

	if err != nil {
		console.ErrorPrintV("Failed to upload file \"%s\" with key \"%s\": %+v", path, hash, err)
		return
	}
}
