package globalcmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/system"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log in.
func LogIn(c *cli.Context) error {
	// Open login link in browser
	// NOTE: Only OAuth (social login) is supported for the CLI due to PKCE.
	port := 4242
	redirectUri := url.QueryEscape(fmt.Sprintf("http://localhost:%d", port))
	authUrl := fmt.Sprintf("%s/login?redirect_uri=%s", config.I.WebsiteURL, redirectUri)
	console.Info("Opening browser to log you in...")
	console.Info("You can also open this URL:")
	fmt.Println(authUrl + "\n")
	system.OpenBrowser(authUrl)

	// Start local HTTP server for receiving authentication redirect requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		console.Verbose("Received auth redirect")

		// Get token (not the session token!) from query params
		sessionToken := r.URL.Query().Get("session_token")
		if sessionToken == "" {
			// User was most likely redirected after successful login, silently exit with non-zero exit code
			os.Exit(0)
		}
		console.Verbose("Session token: %s", sessionToken)

		// Update config with auth data
		console.Verbose("Updating config file with new session...")
		config.I.Auth.SessionToken = sessionToken

		newConfig := config.I
		config.OmitInternalConfig(&newConfig)

		cYaml, err := yaml.Marshal(newConfig)
		if err != nil {
			console.Fatal("Error while marshalling config: %v", err)
		}
		err = ioutil.WriteFile(config.GetConfigPath(), cYaml, 0644)
		if err != nil {
			console.Fatal("Error while writing config: %v", err)
		}

		// Write HTML response
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w,
			`<html>
				<head>
					<meta http-equiv="refresh" content="0; url=%s/login/external/success">
					<title>Redirecting...</title>
				</head>
			</html>`, config.I.WebsiteURL,
		)

		console.Success("Authenticated")
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	// Timeout after 3 minutes
	time.Sleep(time.Second * 180)
	return console.Error("Authentication timed out")
}
