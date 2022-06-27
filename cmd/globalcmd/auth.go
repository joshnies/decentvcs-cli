package globalcmd

import (
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/urfave/cli/v2"
)

// Print authentication status.
func PrintAuthState(c *cli.Context) error {
	// Check if authenticated
	if config.I.Auth.SessionToken == "" {
		return console.Error(constants.ErrMsgNotAuthenticated)
	}

	// Print auth data
	fmt.Println(color.Ize(color.Cyan, "Session token: ") + color.Ize(color.Gray, config.I.Auth.SessionToken))

	return nil
}
