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

type AuthProvider string

const (
	AuthProviderAuth0  AuthProvider = "auth0"
	AuthProviderStytch AuthProvider = "stytch"
)

type VCSStorageConfig struct {
	// Multipart upload part size.
	PartSize int64
	// Workerpool size for parallel file uploads.
	UploadPoolSize int
	// Workerpool size for parallel file downloads.
	DownloadPoolSize int
}

type VCSConfig struct {
	// DecentVCS server hostname.
	ServerHost string
	// Max file size for diffing.
	MaxFileSizeForDiff int64
	// Storage configuration.
	Storage VCSStorageConfig
}

type Config struct {
	// Whether or not to print verbose output.
	Verbose bool
	// Path to the Decent global config file.
	GlobalConfigFilePath string
	AuthProvider         AuthProvider
	// DecentVCS configuration.
	VCS VCSConfig
}

// Singleton CLI config instance.
var I Config

// Initialize the CLI config.
func InitConfig() Config {
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

	var authProvider AuthProvider
	authProviderStr := os.Getenv("AUTH")
	if authProviderStr == "" {
		authProvider = AuthProviderAuth0
	} else if authProviderStr != string(AuthProviderAuth0) && authProviderStr != string(AuthProviderStytch) {
		log.Fatal("Invalid AUTH")
	}

	// Validate VCS config
	vcsServerHost := os.Getenv("VCS_SERVER_HOST")
	if vcsServerHost == "" {
		vcsServerHost = "http://localhost:8080"
	}

	maxFileSizeForDiffStr := os.Getenv("MAX_FILE_SIZE_FOR_DIFF")
	if maxFileSizeForDiffStr == "" {
		maxFileSizeForDiffStr = fmt.Sprint(1 * 1024 * 1024) // 1MB
	}

	maxFileSizeForDiff, err := strconv.ParseInt(maxFileSizeForDiffStr, 10, 64)
	if err != nil {
		log.Fatal("Invalid MAX_FILE_SIZE_FOR_DIFF")
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

	// Construct config
	I = Config{
		Verbose:              os.Getenv(constants.VerboseEnvVar) == "1",
		GlobalConfigFilePath: globalConfigFilePath,
		AuthProvider:         authProvider,
		VCS: VCSConfig{
			ServerHost:         vcsServerHost,
			MaxFileSizeForDiff: maxFileSizeForDiff,
			Storage: VCSStorageConfig{
				PartSize:         partSize,
				UploadPoolSize:   uploadPoolSize,
				DownloadPoolSize: downloadPoolSize,
			},
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
