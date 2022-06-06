package configio

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/models"
)

// Save global config to file.
func SaveGlobalConfig(gc models.GlobalConfig) error {
	// Encode as JSON
	gcJson, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		console.Verbose("Error while encoding global config as JSON: %s", err)
		console.ErrorPrint(constants.ErrMsgInternal)
		os.Exit(1)
	}

	// Create directories if they don't exist
	err = os.MkdirAll(filepath.Dir(config.I.GlobalConfigFilePath), 0755)
	if err != nil {
		return err
	}

	// Write to file, and create it if it doesn't exist
	gcFile, err := os.Create(config.I.GlobalConfigFilePath)
	if err != nil {
		console.Verbose("Error while creating config file: %s", err)
		console.ErrorPrint(constants.ErrMsgInternal)
		os.Exit(1)
	}
	defer gcFile.Close()

	gcFile.Write(gcJson)
	if err != nil {
		console.Verbose("Error while writing config file: %s", err)
		console.ErrorPrint(constants.ErrMsgInternal)
		os.Exit(1)
	}

	return nil
}
