package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/joshnies/decent/models"
)

// Get global Quanta Control config from file.
func GetGlobalConfig() (models.GlobalConfig, error) {
	// Create directories if they don't exist
	err := os.MkdirAll(filepath.Dir(I.GlobalConfigFilePath), 0755)
	if err != nil {
		return models.GlobalConfig{}, err
	}

	// Open file
	gcFile, err := os.Open(I.GlobalConfigFilePath)
	if err != nil {
		return models.GlobalConfig{}, err
	}

	// Decode file contents from JSON
	var gc models.GlobalConfig
	err = json.NewDecoder(gcFile).Decode(&gc)
	if err != nil {
		return models.GlobalConfig{}, err
	}

	return gc, nil
}
