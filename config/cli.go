package config

import "os"

type APIConfig struct {
	// API hostname.
	Host string
}

type StorageConfig struct {
	// Bucket name.
	Bucket string
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
	I = CLIConfig{
		// TODO: Implement sandbox mode.
		Sandbox: os.Getenv("SANDBOX") == "1",
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
