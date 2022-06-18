package auth

import (
	"log"
	"time"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth0"
	"github.com/joshnies/decent/lib/console"
)

// Logs a fatal error if user is not authenticated or their access token has expired
func Validate() {
	// Check if config has auth data
	if config.I.Auth.AccessToken == "" {
		log.Fatal("not authenticated, please run `decent login`")
	}

	// If access token has not yet expired, return it
	if config.I.Auth.AuthenticatedAt+config.I.Auth.ExpiresIn <= time.Now().Unix() {
		// Refresh access token
		console.Verbose("Access token has expired, refreshing...")
		refreshAccessToken()
		console.Verbose("Access token refreshed")
	}
}

// Refresh access token
func refreshAccessToken() {
	// Check if refresh token exists
	if config.I.Auth.RefreshToken == "" {
		// TODO: Handle this error
		log.Fatal("refresh token not found")
	}

	switch config.I.Auth.Provider {
	case config.AuthProviderAuth0:
		auth0.RefreshAccessToken()
	case config.AuthProviderStytch:
		// TODO: Implement
	}
}
