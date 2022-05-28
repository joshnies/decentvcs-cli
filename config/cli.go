package config

import "os"

type APIConfig struct {
	// API hostname.
	Host string
}

type StorageConfig struct {
	// Bucket name.
	Bucket string
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
	I = CLIConfig{
		// TODO: Implement sandbox mode.
		Sandbox: os.Getenv("SANDBOX") == "1",
		Verbose: os.Getenv("VERBOSE") == "1",
		API: APIConfig{
			Host: "http://localhost:8080/v1",
		},
		Storage: StorageConfig{
			Bucket:           "qc-dev",
			PartSize:         5 * 1024 * 1024, // 5MB
			UploadPoolSize:   128,
			DownloadPoolSize: 32,
		},
	}

	return I
}
