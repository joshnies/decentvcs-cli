package vcs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/models"
	"gopkg.in/yaml.v3"
)

// Get the path to the closest DecentVCS project config file (using an upwards file search).
// Returns an error if not found.
func GetProjectConfigPath() (string, error) {
	// Get absolute current directory as initial search path
	searchPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// While config path is empty...
	var configPath string
	for configPath == "" {
		// Check if file exists
		searchPathWithFile := filepath.Join(searchPath, constants.ProjectFileName)
		if _, err := os.Stat(searchPathWithFile); err != nil {
			// If end of search path, return error
			if strings.Split(searchPath, string(os.PathSeparator))[0] == searchPath {
				return "", console.Error(constants.ErrNoProject)
			}

			// Not found yet (or an error occurred), move up one directory
			searchPath = filepath.Dir(searchPath)
		} else {
			// File was found, break
			configPath = searchPathWithFile
			break
		}
	}

	return configPath, nil
}

// Get project config from file in current directory.
func GetProjectConfig() (models.ProjectConfig, error) {
	// Get project config file path
	configPath, err := GetProjectConfigPath()
	if err != nil {
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

	// Validate
	err = ValidateProjectConfig(config)
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
		newConfig = MergeProjectConfigs(exConfig, c)
	}

	// Write
	newConfigBytes, err := yaml.Marshal(newConfig)
	if err != nil {
		return models.ProjectConfig{}, err
	}

	err = ioutil.WriteFile(configPath, newConfigBytes, os.ModePerm)
	return newConfig, err
}

// Validate a project config model.
func ValidateProjectConfig(projectConfig models.ProjectConfig) error {
	v := validator.New()
	return v.Struct(projectConfig)
}

// Merge new project config into the old one.
func MergeProjectConfigs(oldData models.ProjectConfig, newData models.ProjectConfig) models.ProjectConfig {
	merged := oldData

	if newData.ProjectSlug != "" {
		merged.ProjectSlug = newData.ProjectSlug
	}

	if newData.CurrentBranchName != "" {
		merged.CurrentBranchName = newData.CurrentBranchName
	}

	if newData.CurrentCommitIndex != 0 {
		merged.CurrentCommitIndex = newData.CurrentCommitIndex
	}

	return merged
}
