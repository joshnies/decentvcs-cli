package vcs

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/models"
	"gopkg.in/yaml.v3"
)

// Get project config from file in current directory.
// TODO: Find project config file within parent directories
func GetProjectConfig() (models.ProjectConfig, error) {
	// Check if file exists
	configPath := filepath.Join(".", constants.ProjectFileName)
	if _, err := os.Stat(configPath); err != nil {
		return models.ProjectConfig{}, err
	}

	// Read file
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return models.ProjectConfig{}, err
	}

	// Unmarshal
	var config models.ProjectConfig
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return models.ProjectConfig{}, err
	}

	return config, nil
}

// Write project file.
func SaveProjectConfig(path string, c models.ProjectConfig) (models.ProjectConfig, error) {
	configPath := filepath.Join(path, constants.ProjectFileName)

	// Read existing project file (if it exists)
	newConfig := c
	if _, err := os.Stat(configPath); err == nil {
		exBytes, err := ioutil.ReadFile(configPath)
		if err != nil {
			return models.ProjectConfig{}, err
		}

		var exConfig models.ProjectConfig
		err = yaml.Unmarshal(exBytes, &exConfig)
		if err != nil {
			return models.ProjectConfig{}, err
		}

		// Merge existing data with new data
		newConfig = models.MergeProjectConfigs(exConfig, c)
	}

	// Write
	newConfigBytes, err := yaml.Marshal(newConfig)
	if err != nil {
		return models.ProjectConfig{}, err
	}

	err = ioutil.WriteFile(configPath, newConfigBytes, os.ModePerm)
	return newConfig, err
}
