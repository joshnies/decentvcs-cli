package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/joshnies/decent/lib/console"
	"gopkg.in/yaml.v3"
)

type AuthProvider string

const (
	AuthProviderAuth0  AuthProvider = "auth0"
	AuthProviderStytch AuthProvider = "stytch"
)

type VCSStorageConfig struct {
	// Multipart upload part size.
	PartSize int64 `yaml:"part_size"`
	// Workerpool size for parallel file uploads.
	UploadPoolSize int `yaml:"upload_pool_size"`
	// Workerpool size for parallel file downloads.
	DownloadPoolSize int `yaml:"download_pool_size"`
}

type VCSConfig struct {
	// DecentVCS server hostname.
	ServerHost string `yaml:"server_host"`
	// Max file size for diffing.
	MaxFileSizeForDiff int64 `yaml:"max_file_size_for_diff"`
	// Storage configuration.
	Storage VCSStorageConfig
}

type AuthConfig struct {
	// Authentication server provider.
	// Currently supports `auth0` and `stytch`.
	Provider        AuthProvider
	AccessToken     string `yaml:"access_token"`
	RefreshToken    string `yaml:"refresh_token"`
	IDToken         string `yaml:"id_token"`
	ExpiresIn       int64  `yaml:"expires_in"`
	AuthenticatedAt int64  `yaml:"authenticated_at"`
}

type Config struct {
	// Whether or not to print verbose output.
	Verbose bool
	Auth    AuthConfig
	VCS     VCSConfig
}

// Singleton CLI config instance.
var I Config

// Returns path to the Decent global config file.
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(homeDir, ".decent/config.yml")
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
			Verbose: false,
			Auth: AuthConfig{
				Provider: AuthProviderAuth0,
			},
			VCS: VCSConfig{
				ServerHost:         "http://localhost:8080",
				MaxFileSizeForDiff: 1 * 1024 * 1024, // 1 MB
				Storage: VCSStorageConfig{
					// TODO: Should we increase default PartSize to 64 MB?
					PartSize:         5 * 1024 * 1024, // 5 MB
					UploadPoolSize:   128,
					DownloadPoolSize: 32,
				},
			},
		}

		// Write default config to file
		cYaml, err := yaml.Marshal(I)
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(cpath, cYaml, 0644)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Open file
		gcBytes, err := ioutil.ReadFile(cpath)
		if err != nil {
			log.Fatal(err)
		}

		// Decode file contents
		err = yaml.Unmarshal(gcBytes, &I)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Validate config
	if I.VCS.ServerHost == "" {
		log.Fatal("\"vcs.server_host\" must be specified")
	}
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
