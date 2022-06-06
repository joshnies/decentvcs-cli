package global

import (
	"encoding/json"
	"io/ioutil"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
)

// Log out of Quanta Control.
func LogOut(c *cli.Context) error {
	gc := auth.Validate()

	// Clear auth data
	gc.Auth = models.GlobalConfigAuth{}

	// Save global config file
	gcJson, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		console.Verbose("Error while encoding auth data as JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	err = ioutil.WriteFile(config.I.GlobalConfigFilePath, gcJson, 0644)
	if err != nil {
		console.Verbose("Error while writing config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	console.Info("Logged out")
	return nil
}
