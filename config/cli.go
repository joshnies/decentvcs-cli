package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joshnies/quanta/constants"
)

type APIConfig struct {
	// API hostname.
	Host string
}

type StorageConfig struct {
	// Multipart upload part size.
	PartSize int64
	// Workerpool size for parallel file uploads.
	UploadPoolSize int
	// Workerpool size for parallel file downloads.
	DownloadPoolSize int
}

type CLIConfig struct {
	// Whether or not to print verbose output.
	Verbose bool
	// Max file size for diffing.
	MaxFileSizeForDiff int64
	// API configuration.
	API APIConfig
	// Storage configuration.
	Storage StorageConfig
}

// Singleton CLI config instance.
var I CLIConfig

// Initialize the CLI config.
func InitConfig() CLIConfig {
	maxFileSizeForDiffStr := os.Getenv("MAX_FILE_SIZE_FOR_DIFF")
	if maxFileSizeForDiffStr == "" {
		maxFileSizeForDiffStr = fmt.Sprint(1 * 1024 * 1024) // 1MB
	}

	maxFileSizeForDiff, err := strconv.ParseInt(maxFileSizeForDiffStr, 10, 64)
	if err != nil {
		log.Fatal("Invalid MAX_FILE_SIZE_FOR_DIFF")
	}

	partSizeStr := os.Getenv("PART_SIZE")
	if partSizeStr == "" {
		partSizeStr = fmt.Sprint(5 * 1024 * 1024) // 5MB
	}

	partSize, err := strconv.ParseInt(partSizeStr, 10, 64)
	if err != nil {
		log.Fatal("Invalid PART_SIZE")
	}

	uploadPoolSizeStr := os.Getenv("UPLOAD_POOL_SIZE")
	if uploadPoolSizeStr == "" {
		uploadPoolSizeStr = "128"
	}

	uploadPoolSize, err := strconv.Atoi(uploadPoolSizeStr)
	if err != nil {
		log.Fatal("Invalid UPLOAD_POOL_SIZE")
	}

	downloadPoolSizeStr := os.Getenv("DOWNLOAD_POOL_SIZE")
	if downloadPoolSizeStr == "" {
		downloadPoolSizeStr = "32"
	}

	downloadPoolSize, err := strconv.Atoi(downloadPoolSizeStr)
	if err != nil {
		log.Fatal("Invalid DOWNLOAD_POOL_SIZE")
	}

	I = CLIConfig{
		Verbose:            os.Getenv(constants.VerboseEnvVar) == "1",
		MaxFileSizeForDiff: maxFileSizeForDiff,
		API: APIConfig{
			Host: "http://localhost:8080/v1",
		},
		Storage: StorageConfig{
			PartSize:         partSize,
			UploadPoolSize:   uploadPoolSize,
			DownloadPoolSize: downloadPoolSize,
		},
	}

	if I.Verbose {
		// Print config as JSON
		cfgJson, err := json.MarshalIndent(I, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Config:")
		fmt.Println(string(cfgJson))
		fmt.Println()
	}

	return I
}
