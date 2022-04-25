package config

import (
	"encoding/json"
	"os"

	"github.com/joshnies/qc-cli/models"
)

// Get project config from `.qc` file in current directory.
func GetProjectConfig() (models.ProjectFileData, error) {
	// Open `.qc` file
	jsonFile, err := os.Open(".qc")
	if err != nil {
		return models.ProjectFileData{}, err
	}
	defer jsonFile.Close()

	// Decode JSON
	var data models.ProjectFileData
	err = json.NewDecoder(jsonFile).Decode(&data)
	if err != nil {
		return models.ProjectFileData{}, err
	}

	return data, nil
}
