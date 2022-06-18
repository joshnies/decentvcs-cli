package auth0

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/grokify/go-pkce"
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/system"
	"github.com/lucsky/cuid"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log in using Auth0.
func LogIn(c *cli.Context) error {
	// Open login link in browser
	port := 4242
	codeVerifier, err := pkce.NewCodeVerifierWithLength(32)
	if err != nil {
		console.Fatal("Failed to generate code verifier: %v", err)
	}
	codeChallenge := pkce.CodeChallengeS256(codeVerifier)
	serverState := cuid.New()
	cliLocalhost := fmt.Sprintf("http://localhost:%d", port)
	scope := url.QueryEscape("offline_access openid profile email")
	authUrl := constants.Auth0DomainDev + "/authorize?" +
		"response_type=code" +
		"&code_challenge_method=S256" +
		"&code_challenge=" + codeChallenge +
		"&client_id=" + constants.Auth0ClientIDDev +
		"&audience=http://localhost:8080" +
		"&redirect_uri=" + cliLocalhost +
		"&state=" + serverState +
		"&scope=" + scope

	console.Info("Opening browser to log in...")
	console.Info("You can also open this URL:")
	fmt.Println(authUrl)
	system.OpenBrowser(authUrl)

	// Start local HTTP server for receiving Auth0 authentication callback requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		console.Verbose("Received authentication callback request. Validating...")

		code := r.URL.Query().Get("code")
		clientState := r.URL.Query().Get("state")
		resError := r.URL.Query().Get("error")
		resErrorDesc := r.URL.Query().Get("error_description")

		if clientState != serverState {
			console.Verbose("Client state does not match server state")
			console.Verbose("Client state: %s", clientState)
			console.Verbose("Server state: %s", serverState)
			console.Fatal("Auth state check failed")
		}

		if resError != "" {
			console.Fatal(
				"Received error from authentication callback: %s; %s",
				resError,
				resErrorDesc,
			)
		}

		// Validate code
		tokenReqUrl := fmt.Sprintf("%s/oauth/token", constants.Auth0DomainDev)
		tokenReqData := url.Values{}
		tokenReqData.Set("grant_type", "authorization_code")
		tokenReqData.Set("client_id", constants.Auth0ClientIDDev)
		tokenReqData.Set("code_verifier", codeVerifier)
		tokenReqData.Set("code", code)
		tokenReqData.Set("redirect_uri", cliLocalhost)
		tokenRes, err := http.Post(
			tokenReqUrl,
			"application/x-www-form-urlencoded",
			strings.NewReader(tokenReqData.Encode()),
		)
		if err != nil {
			console.Fatal("Error while retrieving access token: %v", err)
		}

		if tokenRes.StatusCode != 200 {
			// Parse response body
			var body map[string]interface{}
			_ = json.NewDecoder(tokenRes.Body).Decode(&body)

			errorDesc := body["error_description"]
			console.Fatal("Received HTTP status %d while retrieving access token: %s", tokenRes.StatusCode, errorDesc)
		}

		// Parse response
		authConfig, err := ParseAccessTokenResponse(tokenRes)
		if err != nil {
			console.Fatal("Error while parsing access token response from Auth0: %v", err)
		}

		// Make sure a refresh token was included in response
		if authConfig.RefreshToken == "" {
			console.Fatal("No refresh token included in response from Auth0")
		}

		// Print auth data
		console.Verbose("Access token: %s", authConfig.AccessToken)
		console.Verbose("Refresh token: %s", authConfig.RefreshToken)
		console.Verbose("ID token: %s", authConfig.IDToken)
		console.Verbose("Expires in: %d hours", authConfig.ExpiresIn/60/60)
		console.Verbose("Authenticated at: %s", authConfig.AuthenticatedAt)

		config.I.Auth = authConfig
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
	return console.Error("Ending authentication attempt after 3 minutes")
}
