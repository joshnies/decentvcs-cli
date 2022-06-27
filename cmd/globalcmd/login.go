package globalcmd

import (
	"fmt"
	"io/ioutil"
	"log"
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
	port := 4242
	redirectUri := url.QueryEscape(fmt.Sprintf("http://localhost:%d", port))
	// TODO: Update authUrl based on env
	authUrl := fmt.Sprintf("http://localhost:3000/login?redirect_uri=%s", redirectUri)
	console.Info("Opening browser to log you in...")
	console.Info("You can also open this URL:")
	fmt.Println(authUrl + "\n")
	system.OpenBrowser(authUrl)

	// Start local HTTP server for receiving authentication redirect requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		console.Verbose("Received auth redirect")

		// Get session token from query params
		token := r.URL.Query().Get("token")
		if token == "" {
			log.Fatal("Request received, but no token found")
		}

		console.Verbose("Session token (?): %s", token)
		console.Verbose("Updating config file with new session...")

		// Update config with auth data
		config.I.Auth.SessionToken = token
		cYaml, err := yaml.Marshal(config.I)
		if err != nil {
			console.Fatal("Error while marshalling config: %v", err)
		}
		err = ioutil.WriteFile(config.GetConfigPath(), cYaml, 0644)
		if err != nil {
			console.Fatal("Error while writing config: %v", err)
		}

		// TODO: Write HTML response

		console.Success("Authenticated")
		os.Exit(0)
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	// Timeout after 3 minutes
	time.Sleep(time.Second * 180)
	return console.Error("Authentication timed out")
}
