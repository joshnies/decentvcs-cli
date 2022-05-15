package cmd

import (
	"fmt"
	"time"

	"github.com/TwiN/go-color"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
	"github.com/urfave/cli/v2"
)

// Print authentication status.
func PrintAuthState(c *cli.Context) error {
	// Get global config
	gc, err := config.GetGlobalConfig()
	if err != nil {
		return err
	}

	// Check if authenticated
	if gc.Auth.AccessToken == "" {
		return console.Error("Not logged in yet. Use `qc login` to log in.")
	}

	// Print auth data
	authAtf := time.Unix(gc.Auth.AuthenticatedAt, 0).Format(constants.TimeFormat)
	fmt.Println(color.Ize(color.Cyan, "Access token: ") + color.Ize(color.Gray, gc.Auth.AccessToken))
	fmt.Println(color.Ize(color.Cyan, "Refresh token: ") + color.Ize(color.Gray, gc.Auth.RefreshToken))
	fmt.Println(color.Ize(color.Cyan, "ID token: ") + color.Ize(color.Gray, gc.Auth.IDToken))
	fmt.Println(color.Ize(color.Cyan, "Authenticated at: "), color.Ize(color.Gray, authAtf))

	expiresAt := time.Unix(gc.Auth.AuthenticatedAt, 0).Add(time.Duration(gc.Auth.ExpiresIn) * time.Second)
	fmt.Println(color.Ize(color.Cyan, "Expires at: ") + color.Ize(color.Gray, expiresAt.Format(constants.TimeFormat)))

	expiresInHours := time.Until(expiresAt).Truncate(time.Second)
	fmt.Println(color.Ize(color.Cyan, "Expires in: ") + color.Ize(color.Gray, expiresInHours.String()))

	return nil
}
