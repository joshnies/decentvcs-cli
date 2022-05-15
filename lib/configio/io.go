package configio

import (
	"encoding/json"
	"os"

	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/models"
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

	// Write to file, and create it if it doesn't exist
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		console.Verbose("Error while retrieving user home directory: %s", err)
		console.ErrorPrint(constants.ErrMsgInternal)
		os.Exit(1)
	}

	gcFile, err := os.Create(userHomeDir + "/" + constants.GlobalConfigFileName)
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
