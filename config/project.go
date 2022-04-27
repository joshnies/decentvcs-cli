package config

import (
	"encoding/json"
	"os"

	"github.com/joshnies/qc-cli/models"
)

// Get project config from `.qc` file in current directory.
func GetProjectConfig() (models.ProjectConfig, error) {
	// Open `.qc` file
	jsonFile, err := os.Open(".qc")
	if err != nil {
		return models.ProjectConfig{}, err
	}
	defer jsonFile.Close()

	// Decode JSON
	var data models.ProjectConfig
	err = json.NewDecoder(jsonFile).Decode(&data)
	if err != nil {
		return models.ProjectConfig{}, err
	}

	return data, nil
}
