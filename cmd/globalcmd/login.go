package globalcmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/system"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log in.
func LogIn(c *cli.Context) error {
	// Open login link in browser
	port := 4242
	relayUri := url.QueryEscape(fmt.Sprintf("http://localhost:%d", port))
	// TODO: Update authUrl based on env
	authUrl := fmt.Sprintf("http://localhost:3000/login?relay=%s", relayUri)
	console.Info("Opening browser to log you in...")
	console.Info("You can also open this URL:")
	fmt.Println(authUrl + "\n")
	system.OpenBrowser(authUrl)

	// Start local HTTP server (a.k.a. the "relay") for receiving authentication callback requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		console.Verbose("Validating authentication callback request...")

		// Parse response
		var res models.LoginRelayRes
		err := json.NewDecoder(r.Body).Decode(&res)
		if err != nil {
			console.Error("Error parsing relay response: %s", err)
			return
		}

		// Print auth data
		console.Verbose("Session token: %s", res.SessionToken)

		// Update config with auth data
		config.I.Auth.SessionToken = res.SessionToken
		cYaml, err := yaml.Marshal(config.I)
		if err != nil {
			console.Fatal("Error while marshalling config: %v", err)
		}
		err = ioutil.WriteFile(config.GetConfigPath(), cYaml, 0644)
		if err != nil {
			console.Fatal("Error while writing config: %v", err)
		}

		console.Success("Authenticated")
		os.Exit(0)
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	// Timeout after 3 minutes
	time.Sleep(time.Second * 180)
	return console.Error("Authentication timed out")
}
