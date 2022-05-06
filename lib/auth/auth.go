package auth

import (
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
)

// Returns true if the user is logged in, false otherwise.
func Validate() {
	// Get global config
	gc, err := config.GetGlobalConfig()
	if err != nil || gc.Auth.AccessToken == "" {
		console.Fatal(constants.ErrMsgNotAuthenticated)
	}
}
