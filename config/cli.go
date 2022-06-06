package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joshnies/decent/constants"
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

type CLIVCSConfig struct {
	// Max file size for diffing.
	MaxFileSizeForDiff int64
}

type CLIConfig struct {
	// Whether or not to print verbose output.
	Verbose bool
	// Path to the Decent global config file.
	GlobalConfigFilePath string
	// API configuration.
	API APIConfig
	// Storage configuration.
	Storage StorageConfig
	// DecentVCS configuration.
	VCS CLIVCSConfig
}

// Singleton CLI config instance.
var I CLIConfig

// Initialize the CLI config.
func InitConfig() CLIConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Validate root config
	globalConfigFilePath := os.Getenv("DECENT_CONFIG")
	if globalConfigFilePath == "" {
		globalConfigFilePath = filepath.Join(homeDir, ".decent/config.json")
	} else {
		globalConfigFilePath = strings.Replace(globalConfigFilePath, "~", homeDir, 1)
	}

	// Validate storage config
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

	// Validate VCS config
	maxFileSizeForDiffStr := os.Getenv("MAX_FILE_SIZE_FOR_DIFF")
	if maxFileSizeForDiffStr == "" {
		maxFileSizeForDiffStr = fmt.Sprint(1 * 1024 * 1024) // 1MB
	}

	maxFileSizeForDiff, err := strconv.ParseInt(maxFileSizeForDiffStr, 10, 64)
	if err != nil {
		log.Fatal("Invalid MAX_FILE_SIZE_FOR_DIFF")
	}

	// Construct config
	I = CLIConfig{
		Verbose:              os.Getenv(constants.VerboseEnvVar) == "1",
		GlobalConfigFilePath: globalConfigFilePath,
		API: APIConfig{
			Host: "http://localhost:8080/v1",
		},
		Storage: StorageConfig{
			PartSize:         partSize,
			UploadPoolSize:   uploadPoolSize,
			DownloadPoolSize: downloadPoolSize,
		},
		VCS: CLIVCSConfig{
			MaxFileSizeForDiff: maxFileSizeForDiff,
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
