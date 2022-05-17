package storage

import (
	"context"
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
	chunks := util.ChunkMap(hashMap, 8)

	// For each chunk...
	for _, chunk := range chunks {
		// Setup for parallel uploads
		var wg sync.WaitGroup
		wg.Add(len(maps.Keys(chunk)))

		p := mpb.New(mpb.WithWidth(60))

		// Upload objects in parallel
		for path, key := range hashMap {
			go uploadRoutine(ctx, client, path, key, p)
		}

		// Wait for uploads to finish
		wg.Wait()
		p.Wait()
	}

	return nil
}

func uploadRoutine(ctx context.Context, client *s3.Client, local string, remote string, p *mpb.Progress) {
	// Read local file
	file, err := os.Open(local)
	if err != nil {
		console.ErrorPrintV("Failed to open file \"%s\": %v", local, err)
		return
	}

	// Get local file size
	fileInfo, err := file.Stat()
	if err != nil {
		console.ErrorPrint("Failed to stat file \"%s\": %v", local, err)
		panic(err)
	}

	total := fileInfo.Size()

	// Add progress bar
	bar := p.AddBar(total,
		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f "),
		),
		mpb.AppendDecorators(
			decor.Name(filepath.Base(local), decor.WC{W: 20, C: decor.DidentRight}),
			decor.Name(" | "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)
	proxyReader := bar.ProxyReader(file)
	defer proxyReader.Close()

	// Upload new object to S3
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &config.I.Storage.Bucket,
		Key:    &remote,
		Body:   proxyReader,
	})

	if err != nil {
		console.ErrorPrintV("Failed to upload file \"%s\" with key \"%s\": %+v", local, remote, err)
		return
	}
}
