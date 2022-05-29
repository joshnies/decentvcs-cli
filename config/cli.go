package config

import (
	"log"
	"os"
	"strconv"
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
	// Whether or not to run in sandbox mode.
	Sandbox bool
	// Whether or not to print verbose output.
	Verbose bool
	// API configuration.
	API APIConfig
	// Storage configuration.
	Storage StorageConfig
}

// Singleton CLI config instance.
var I CLIConfig

// Initialize the CLI config.
func InitConfig() CLIConfig {
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
		// TODO: Implement sandbox mode.
		Sandbox: os.Getenv("SANDBOX") == "1",
		Verbose: os.Getenv("VERBOSE") == "1",
		API: APIConfig{
			Host: "http://localhost:8080/v1",
		},
		Storage: StorageConfig{
			PartSize:         5 * 1024 * 1024, // 5MB
			UploadPoolSize:   uploadPoolSize,
			DownloadPoolSize: downloadPoolSize,
		},
	}

	return I
}
