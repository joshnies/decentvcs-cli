package config

import (
	"encoding/json"
	"os"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/models"
)

// Get global Quanta Control config from file.
func GetGlobalConfig() (models.GlobalConfig, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return models.GlobalConfig{}, err
	}

	gcPath := userHomeDir + "/" + constants.GlobalConfigFileName
	gcFile, err := os.Open(gcPath)
	if err != nil {
		return models.GlobalConfig{}, err
	}

	var gc models.GlobalConfig
	err = json.NewDecoder(gcFile).Decode(&gc)
	if err != nil {
		return models.GlobalConfig{}, err
	}

	return gc, nil
}
