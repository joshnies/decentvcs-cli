package global

import (
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth0"
	"github.com/urfave/cli/v2"
)

// Log in.
func LogIn(c *cli.Context) error {
	switch config.I.AuthProvider {
	case config.AuthProviderAuth0:
		return auth0.LogIn(c)
	case config.AuthProviderStytch:
		// TODO: Implement Stytch login
		return nil
	}

	return nil
}
