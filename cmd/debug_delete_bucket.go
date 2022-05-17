package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
	"github.com/urfave/cli/v2"
)

// DEBUG
// Delete bucket
func DebugDeleteBucket(c *cli.Context) error {
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

	// List objects in bucket
	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &config.I.Storage.Bucket,
	})
	if err != nil {
		return err
	}

	// Add keys to delete list
	keys := []string{}
	for _, item := range out.Contents {
		keys = append(keys, *item.Key)
	}

	// Delete objects
	for _, key := range keys {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &config.I.Storage.Bucket,
			Key:    &key,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
