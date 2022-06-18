package globalcmd

import (
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth0"
	"github.com/joshnies/decent/lib/console"
	"github.com/urfave/cli/v2"
)

// Log in.
func LogIn(c *cli.Context) error {
	switch config.I.Auth.Provider {
	case config.AuthProviderAuth0:
		return auth0.LogIn(c)
	case config.AuthProviderStytch:
		// TODO: Implement Stytch login
		return nil
	}

	return console.Error(
		"Invalid authentication provider \"%s\". Must be either \"%s\" or \"%s\".",
		config.I.Auth.Provider,
		config.AuthProviderAuth0,
		config.AuthProviderStytch,
	)
}
