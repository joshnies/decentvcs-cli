package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/decentvcs/cli/lib/console"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

type Env string

const (
	// Local environment
	EnvLcl Env = "lcl"
	// Development environment
	EnvDev Env = "dev"
	// Production environment
	EnvPrd Env = "prd"
)

type VCSStorageConfig struct {
	// Multipart upload part size.
	PartSize int64 `yaml:"part_size"`
	// Workerpool size for parallel file uploads.
	UploadPoolSize int `yaml:"upload_pool_size"`
	// Workerpool size for parallel file downloads.
	DownloadPoolSize int `yaml:"download_pool_size"`
	// Amount of objects in a single chunk to presign in parallel.
	PresignChunkSize int `yaml:"presign_chunk_size"`
}

type VCSConfig struct {
	// DecentVCS server hostname.
	ServerHost string `yaml:",omitempty"`
	// Max file size for diffing.
	MaxFileSizeForDiff int64 `yaml:"max_file_size_for_diff"`
	// Storage configuration.
	Storage VCSStorageConfig
}

type AuthConfig struct {
	SessionToken string `yaml:"session_token,omitempty"`
}

type Config struct {
	// Environment to run the CLI in.
	Env Env `yaml:",omitempty"`
	// Whether or not to print verbose output.
	Verbose bool
	//
	// [Internal]
	//
	// DecentVCS website URL.
	WebsiteURL string `yaml:",omitempty"`
	Auth       AuthConfig
	VCS        VCSConfig
	// Rate limiter for uploading/downloading files to or from storage.
	// Required to abide by rate limits set by storage providers.
	RateLimiter *rate.Limiter
}

// Singleton CLI config instance.
var I Config

// Returns path to the DecentVCS global config file.
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(homeDir, ".decent/config.yml")
}

// Returns the dashboard URL based on the CLI environment.
func getDashURL(env Env) string {
	switch env {
	case EnvDev:
		return "http://dev.app.decentvcs.com"
	case EnvLcl:
		return "http://localhost:3000"
	default:
		// Production is the default
		return "https://app.decentvcs.com"
	}
}

// Returns the DecentVCS server host based on the CLI environment.
func getVCSServerHost(env Env) string {
	switch env {
	case EnvDev:
		return "http://vcs-dev.decentvcs.com"
	case EnvLcl:
		return "http://localhost:8080"
	default:
		// Production is the default
		return "https://vcs.decentvcs.com"
	}
}

// Initialize the CLI config.
func InitConfig() Config {
	cpath := GetConfigPath()

	// Create default config file if it doesn't exist yet
	if _, err := os.Stat(cpath); errors.Is(err, os.ErrNotExist) {
		// Create directories if they don't exist
		err := os.MkdirAll(filepath.Dir(cpath), 0755)
		if err != nil {
			log.Fatal(err)
		}

		I = Config{
			VCS: VCSConfig{
				MaxFileSizeForDiff: 1 * 1024 * 1024, // 1 MB
				Storage: VCSStorageConfig{
					PartSize:         64 * 1024 * 1024, // 64 MB
					UploadPoolSize:   32,
					DownloadPoolSize: 32,
				},
			},
		}

		// Write default config to file
		cYaml, err := yaml.Marshal(I)
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(cpath, cYaml, 0644)
		if err != nil {
			log.Fatal(err)
		}

		// Set internal and default config fields
		SetInternalConfigFields(&I)
	} else {
		// Open file
		gcBytes, err := os.ReadFile(cpath)
		if err != nil {
			log.Fatal(err)
		}

		// Decode file contents
		var config Config
		err = yaml.Unmarshal(gcBytes, &config)
		if err != nil {
			log.Fatal(err)
		}

		// Set internal and default config fields
		SetInternalConfigFields(&config)

		// Set config instance
		I = config
	}

	// Validate config
	if I.VCS.MaxFileSizeForDiff == 0 {
		log.Fatal("\"vcs.max_file_size_for_diff\" must be specified")
	}
	if I.VCS.Storage.PartSize == 0 {
		log.Fatal("\"vcs.storage.part_size\" must be specified")
	}
	if I.VCS.Storage.UploadPoolSize == 0 {
		log.Fatal("\"vcs.storage.upload_pool_size\" must be specified")
	}
	if I.VCS.Storage.DownloadPoolSize == 0 {
		log.Fatal("\"vcs.storage.download_pool_size\" must be specified")
	}

	if I.Verbose {
		// Print config as JSON
		cfgJson, err := json.MarshalIndent(I, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		console.Verbose("Config:")
		console.Verbose(string(cfgJson))
	}

	return I
}

// Set internal config fields.
func SetInternalConfigFields(config *Config) {
	// Set defaults for missing fields
	if config.Env == "" {
		config.Env = EnvPrd
	}

	// Set internal config fields
	config.WebsiteURL = getDashURL(config.Env)
	config.VCS.ServerHost = getVCSServerHost(config.Env)
	config.VCS.Storage.PresignChunkSize = 1024
	config.RateLimiter = rate.NewLimiter(rate.Every(time.Second/90), 1) // TODO: Make this the max of 100 RPS if possible
}

// Omit internal config fields from a config object.
// This should always be called before writing it to a file.
func OmitInternalConfig(config *Config) {
	// Remove internal config fields
	config.WebsiteURL = ""
	config.VCS.Storage.PresignChunkSize = 0
	config.VCS.ServerHost = ""
	config.RateLimiter = nil
}
