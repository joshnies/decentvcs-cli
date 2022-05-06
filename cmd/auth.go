package cmd

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
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/system"
	"github.com/joshnies/qc-cli/models"
	"github.com/lucsky/cuid"
	"github.com/urfave/cli/v2"
)

// Log in to Quanta Control.
func LogIn(c *cli.Context) error {
	// Reference: https://www.altostra.com/blog/cli-authentication-with-auth0
	console.Info("Opening browser to log in to Quanta Control...")

	// Open login link in browser
	codeVerifier, err := pkce.NewCodeVerifierWithLength(32)
	if err != nil {
		console.Verbose("Failed to generate code verifier: %v", err)
		console.ErrorPrint(constants.ErrMsgInternal)
		os.Exit(1)
	}
	codeChallenge := pkce.CodeChallengeS256(codeVerifier)
	serverState := cuid.New()
	localhost := "http://localhost:4242"
	scope := url.QueryEscape("offline_access openid profile email")
	system.OpenBrowser(
		constants.Auth0DomainDev + "/authorize?" +
			"response_type=code" +
			"&code_challenge_method=S256" +
			"&code_challenge=" + codeChallenge +
			"&client_id=" + constants.Auth0ClientIDDev +
			"&audience=" + localhost +
			"&redirect_uri=" + localhost +
			"&state=" + serverState +
			"&scope=" + scope,
	)

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
			console.ErrorPrint(constants.ErrMsgAuthFailed)
			os.Exit(1)
		}

		if resError != "" {
			console.Verbose(
				"Received error from authentication callback: %s; %s",
				resError,
				resErrorDesc,
			)
			console.ErrorPrint(constants.ErrMsgAuthFailed)
			os.Exit(1)
		}

		// Validate code
		tokenReqUrl := fmt.Sprintf("%s/oauth/token", constants.Auth0DomainDev)
		tokenReqData := url.Values{}
		tokenReqData.Set("grant_type", "authorization_code")
		tokenReqData.Set("client_id", constants.Auth0ClientIDDev)
		tokenReqData.Set("code_verifier", codeVerifier)
		tokenReqData.Set("code", code)
		tokenReqData.Set("redirect_uri", localhost)
		tokenRes, err := http.Post(
			tokenReqUrl,
			"application/x-www-form-urlencoded",
			strings.NewReader(tokenReqData.Encode()),
		)
		if err != nil {
			console.Verbose("Error while retrieving access token: %s", err)
			console.ErrorPrint(constants.ErrMsgAuthFailed)
			os.Exit(1)
		}

		if tokenRes.StatusCode != 200 {
			console.Verbose("Error while retrieving access token: %s", tokenRes.Status)
			console.ErrorPrint(constants.ErrMsgAuthFailed)
			os.Exit(1)
		}

		// Parse response
		var tokenResData map[string]interface{}
		err = json.NewDecoder(tokenRes.Body).Decode(&tokenResData)
		if err != nil {
			console.Verbose("Error while parsing access token response: %s", err)
			console.ErrorPrint(constants.ErrMsgAuthFailed)
			os.Exit(1)
		}

		expiresIn := int(tokenResData["expires_in"].(float64))

		console.Verbose("Access token: %s", tokenResData["access_token"])
		console.Verbose("Refresh token: %s", tokenResData["refresh_token"])
		console.Verbose("ID token: %s", tokenResData["id_token"])
		console.Verbose("Expires in: %d hours", expiresIn/60/60)

		// Save auth data to global config file
		gc := models.GlobalConfig{
			Auth: models.GlobalConfigAuth{
				AccessToken:     tokenResData["access_token"].(string),
				RefreshToken:    tokenResData["refresh_token"].(string),
				IDToken:         tokenResData["id_token"].(string),
				ExpiresIn:       expiresIn,
				AuthenticatedAt: time.Now().Unix(),
			},
		}

		gcJson, err := json.MarshalIndent(gc, "", "  ")
		if err != nil {
			console.Verbose("Error while encoding auth data as JSON: %s", err)
			console.ErrorPrint(constants.ErrMsgInternal)
			os.Exit(1)
		}

		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			console.Verbose("Error while retrieving user home directory: %s", err)
			console.ErrorPrint(constants.ErrMsgInternal)
			os.Exit(1)
		}

		gcFile, err := os.Create(userHomeDir + "/" + constants.GlobalConfigFileName)
		if err != nil {
			console.Verbose("Error while creating config file: %s", err)
			console.ErrorPrint(constants.ErrMsgInternal)
			os.Exit(1)
		}
		defer gcFile.Close()

		gcFile.Write(gcJson)
		if err != nil {
			console.Verbose("Error while writing config file: %s", err)
			console.ErrorPrint(constants.ErrMsgInternal)
			os.Exit(1)
		}

		console.Info("Authentication successful")
		os.Exit(0)
	})
	go http.ListenAndServe(":4242", nil)

	// Timeout after 3 minutes
	time.Sleep(time.Second * 180)
	return console.Error("Ending authentication process after 3 minutes")
}

// Log out of Quanta Control.
func LogOut(c *cli.Context) error {
	// Read existing global config file
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		console.Verbose("Error while retrieving user home directory: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	gcPath := userHomeDir + "/" + constants.GlobalConfigFileName
	gcFile, err := os.Open(gcPath)
	if err != nil {
		console.Verbose("Error while opening config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	var gc models.GlobalConfig
	err = json.NewDecoder(gcFile).Decode(&gc)
	if err != nil {
		console.Verbose("Error while decoding config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	// Return if not authenticated
	if gc.Auth.AccessToken == "" {
		return console.Error(constants.ErrMsgNotAuthenticated)
	}

	// Clear auth data
	gc.Auth = models.GlobalConfigAuth{}

	// Save global config file
	gcJson, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		console.Verbose("Error while encoding auth data as JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	err = ioutil.WriteFile(gcPath, gcJson, 0644)
	if err != nil {
		console.Verbose("Error while writing config file: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	console.Info("Logged out")

	return nil
}

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
	console.Info("Access token: %s", gc.Auth.AccessToken)
	console.Info("Refresh token: %s", gc.Auth.RefreshToken)
	console.Info("ID token: %s", gc.Auth.IDToken)
	console.Info("Authenticated at: %s", time.Unix(gc.Auth.AuthenticatedAt, 0).Format(constants.TimeFormat))

	expiresAt := time.Unix(gc.Auth.AuthenticatedAt, 0).Add(time.Duration(gc.Auth.ExpiresIn) * time.Second)
	console.Info("Expires at: %s", expiresAt.Format(constants.TimeFormat))

	expiresInHours := time.Until(expiresAt).Truncate(time.Second)
	console.Info("Expires in: %s", expiresInHours)

	return nil
}
