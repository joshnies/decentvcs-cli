package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/models"
)

// Get project config from `.qc` file in current directory.
func GetProjectConfig() (models.ProjectConfig, error) {
	// Open `.qc` file
	// TODO: Find project config file within parent directories?
	jsonFile, err := os.Open(".qc")
	if err != nil {
		return models.ProjectConfig{}, console.Error(constants.ErrMsgNoProject)
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

// Write project file.
func SaveProjectConfig(path string, data models.ProjectConfig) (models.ProjectConfig, error) {
	configPath := filepath.Join(path, constants.ProjectFileName)

	// Read existing project file (if it exists)
	jsonFile, err := os.Open(configPath)
	if err != nil && !os.IsNotExist(err) {
		return models.ProjectConfig{}, err
	}
	defer jsonFile.Close()

	// Decode existing data JSON
	var existingData models.ProjectConfig
	if jsonFile != nil {
		err = json.NewDecoder(jsonFile).Decode(&existingData)
		if err != nil {
			// TODO: Improve this error
			return models.ProjectConfig{}, err
		}
	}

	// If existing data exists, merge it with new data
	mergedData := models.MergeProjectConfigs(existingData, data)

	// Write
	json, err := json.MarshalIndent(mergedData, "", "  ")
	if err != nil {
		// TODO: Improve this error
		return models.ProjectConfig{}, err
	}

	err = ioutil.WriteFile(configPath, json, os.ModePerm)
	return mergedData, err
}
