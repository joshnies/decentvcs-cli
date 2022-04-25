package config

import "os"

type APIConfig struct {
	Host string
}

type CLIConfig struct {
	Debug bool
	API   APIConfig
}

var I CLIConfig

// Initialize the CLI config.
func InitConfig() CLIConfig {
	I = CLIConfig{
		Debug: os.Getenv("DEBUG") == "true",
		API: APIConfig{
			Host: os.Getenv("API_HOST"),
		},
	}

	return I
}
