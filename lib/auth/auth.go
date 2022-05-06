package auth

import (
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/models"
)

// Returns true if the user is logged in, false otherwise.
func Validate() models.GlobalConfig {
	// Get global config
	gc, err := config.GetGlobalConfig()

	// TODO: Check for expiration and refresh the access token
	if err != nil || gc.Auth.AccessToken == "" {
		console.Fatal(constants.ErrMsgNotAuthenticated)
	}

	return gc
}
