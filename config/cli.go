package config

import "os"

type APIConfig struct {
	Host string
}

type StorageConfig struct {
	Bucket string
}

type CLIConfig struct {
	Verbose bool
	API     APIConfig
	Storage StorageConfig
}

var I CLIConfig

// Initialize the CLI config.
func InitConfig() CLIConfig {
	// TODO: Update to production values once ready, with env var overrides for testing
	I = CLIConfig{
		Verbose: os.Getenv("VERBOSE") == "1",
		API: APIConfig{
			Host: "http://localhost:8080/v1",
		},
		Storage: StorageConfig{
			Bucket: "qc-dev",
		},
	}

	return I
}
