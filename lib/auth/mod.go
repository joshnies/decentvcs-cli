package auth

import (
	"log"

	"github.com/decentvcs/cli/config"
)

// Logs a fatal error if the user not not have an existing auth token for DecentVCS.
func HasToken() {
	// Check if config has auth data
	if config.I.Auth.SessionToken == "" {
		log.Fatal("not authenticated, please run `decent login`")
	}
}
