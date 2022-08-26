package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/system"
	"github.com/decentvcs/cli/models"
	"github.com/go-playground/validator/v10"
	"github.com/rs/cors"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log in.
func LogIn(c *cli.Context) error {
	// Open login link in browser
	// NOTE: Only OAuth (social login) is supported for the CLI due to PKCE.
	port := 4242
	webhookURL := url.QueryEscape(fmt.Sprintf("http://localhost:%d", port))
	authUrl := fmt.Sprintf("%s/login?require=true&webhook=%s", config.I.WebsiteURL, webhookURL)
	console.Info("Opening browser to log you in...")
	console.Info("You can also open this URL:")
	fmt.Println(authUrl + "\n")
	system.OpenBrowser(authUrl)

	// Start local HTTP server for receiving authentication webhook requests
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{config.I.WebsiteURL},
		AllowedMethods: []string{"POST"},
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		console.Verbose("Received request to auth webhook")

		// Parse request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			console.Error("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Parse request body
		var data models.AuthWebhookRequest
		err = json.Unmarshal(body, &data)
		if err != nil {
			console.Error("Failed to parse request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Validate request body
		validate := validator.New()
		if err := validate.Struct(data); err != nil {
			console.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		console.Verbose("Session token: %s", data.SessionToken)

		// Update config with auth data
		console.Verbose("Updating config file with new session...")
		config.I.Auth.SessionToken = data.SessionToken

		newConfig := config.I
		config.OmitInternalConfig(&newConfig)

		cYaml, err := yaml.Marshal(newConfig)
		if err != nil {
			console.Fatal("Error while marshalling config: %v", err)
		}
		err = os.WriteFile(config.GetConfigPath(), cYaml, 0644)
		if err != nil {
			console.Fatal("Error while writing config: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		console.Success("Authenticated")
		os.Exit(0)
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", port), corsMiddleware.Handler(handler))

	// Timeout after 3 minutes
	time.Sleep(time.Second * 180)
	return console.Error("Authentication timed out")
}
