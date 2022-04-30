package cmd

import (
	"net/http"
	"os"
	"time"

	"github.com/joshnies/qc-cli/lib/console"
	"github.com/urfave/cli/v2"
)

// Log in to Quanta Control.
func LogIn(c *cli.Context) error {
	console.Info("Opening browser to log in to Quanta Control...")

	// Start local HTTP server for receiving Auth0 authentication callback requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Handle callback request
		// Reference: https://www.altostra.com/blog/cli-authentication-with-auth0
		console.Info("Received authentication callback request")
		os.Exit(0)
	})
	go http.ListenAndServe(":4242", nil)

	// Timeout after 3 minutes
	time.Sleep(time.Second * 180)
	return console.Error("Ending authentication process after 3 minutes")
}
