package config

import "os"

type APIConfig struct {
	Host string
}

type CLIConfig struct {
	Verbose bool
	API     APIConfig
}

var I CLIConfig

// Initialize the CLI config.
func InitConfig() CLIConfig {
	I = CLIConfig{
		Verbose: os.Getenv("VERBOSE") == "1",
		API: APIConfig{
			Host: "http://localhost:8080/v1",
		},
	}

	return I
}
