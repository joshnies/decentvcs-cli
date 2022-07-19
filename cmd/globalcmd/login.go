package globalcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/system"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log in.
func LogIn(c *cli.Context) error {
	// Open login link in browser
	// NOTE: Only OAuth (social login) is supported for the CLI due to PKCE.
	port := 4242
	redirectUri := url.QueryEscape(fmt.Sprintf("http://localhost:%d", port))
	authUrl := fmt.Sprintf("%s/login?require=true&require_oauth=true&redirect_uri=%s", config.I.WebsiteURL, redirectUri)
	console.Info("Opening browser to log you in...")
	console.Info("You can also open this URL:")
	fmt.Println(authUrl + "\n")
	system.OpenBrowser(authUrl)

	// Start local HTTP server for receiving authentication redirect requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		console.Verbose("Received auth redirect")

		// Get token (not the session token!) from query params
		token := r.URL.Query().Get("token")
		if token == "" {
			// User was most likely redirected after successful login, silently exit with non-zero exit code
			os.Exit(0)
		}
		console.Verbose("Token: %s", token)

		tokenType := r.URL.Query().Get("stytch_token_type")
		if tokenType == "" {
			// User was most likely redirected after successful login, silently exit with non-zero exit code
			os.Exit(0)
		}
		console.Verbose("Token type: %s", tokenType)

		// Authenticate with DecentVCS server
		console.Verbose("Authenticating with DecentVCS server...")
		httpClient := http.Client{}
		reqUrl := config.I.VCS.ServerHost + "/authenticate"
		reqBody := models.AuthenticateRequest{
			Token:     token,
			TokenType: tokenType,
		}
		reqBodyJson, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", reqUrl, bytes.NewBuffer(reqBodyJson))
		req.Header.Set("Content-Type", "application/json")
		res, err := httpClient.Do(req)
		if err != nil {
			console.Fatal("Failed to authenticate with DecentVCS server: %s", err.Error())
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			console.Fatal("Failed to authenticate with DecentVCS server: %s", err.Error())
		}
		defer res.Body.Close()

		// Parse authentication response
		console.Verbose("Parsing DecentVCS server authentication response...")
		var authRes models.AuthenticateResponse
		err = json.NewDecoder(res.Body).Decode(&authRes)
		if err != nil {
			console.Fatal("Failed to parse DecentVCS server authentication response: %s", err.Error())
		}

		if authRes.SessionToken == "" {
			console.Fatal("Failed to authenticate with DecentVCS server: no session token returned")
		}

		// Update config with auth data
		console.Verbose("Updating config file with new session...")
		config.I.Auth.SessionToken = authRes.SessionToken
		cYaml, err := yaml.Marshal(config.I)
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
