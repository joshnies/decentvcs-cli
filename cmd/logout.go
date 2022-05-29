package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/joshnies/quanta-cli/constants"
	"github.com/joshnies/quanta-cli/lib/console"
	"github.com/joshnies/quanta-cli/models"
	"github.com/urfave/cli/v2"
)

// Log out of Quanta Control.
func LogOut(c *cli.Context) error {
	// Read existing global config file
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		console.Verbose("Error while retrieving user home directory: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	gcPath := userHomeDir + "/" + constants.GlobalConfigFileName
	gcFile, err := os.Open(gcPath)
	if err != nil {
		console.Verbose("Error while opening config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	var gc models.GlobalConfig
	err = json.NewDecoder(gcFile).Decode(&gc)
	if err != nil {
		console.Verbose("Error while decoding config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	// Return if not authenticated
	if gc.Auth.AccessToken == "" {
		return console.Error(constants.ErrMsgNotAuthenticated)
	}

	// Clear auth data
	gc.Auth = models.GlobalConfigAuth{}

	// Save global config file
	gcJson, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		console.Verbose("Error while encoding auth data as JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	err = ioutil.WriteFile(gcPath, gcJson, 0644)
	if err != nil {
		console.Verbose("Error while writing config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	console.Info("Logged out")
	return nil
}
