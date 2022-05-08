package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/configio"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/models"
)

// Returns true if the user is logged in, false otherwise.
func Validate() models.GlobalConfig {
	// Get global config
	gc, err := config.GetGlobalConfig()

	// TODO: Check for expiration and refresh the access token
	if err != nil || gc.Auth.AccessToken == "" {
		console.Fatal(constants.ErrMsgNotAuthenticated)
	}

	return gc
}

// Parse access token response from Auth0 Authentication API.
func ParseAccessTokenResponse(res *http.Response) (models.GlobalConfigAuth, error) {
	// Parse response
	var authConfig models.GlobalConfigAuth
	err := json.NewDecoder(res.Body).Decode(&authConfig)
	if err != nil {
		return models.GlobalConfigAuth{}, err
	}

	// Extract vars from response
	if authConfig.AccessToken == "" {
		return models.GlobalConfigAuth{}, console.Error("Access token not found in response")
	}

	if authConfig.RefreshToken == "" {
		return models.GlobalConfigAuth{}, console.Error("Refresh token not found in response")
	}

	if authConfig.IDToken == "" {
		return models.GlobalConfigAuth{}, console.Error("ID token not found in response")
	}

	if authConfig.ExpiresIn == 0 {
		return models.GlobalConfigAuth{}, console.Error("Expiration (expires_in) not found in response")
	}

	// Add additional data
	authConfig.AuthenticatedAt = time.Now().Unix()
	return authConfig, nil
}

// Refresh access token
func RefreshAccessToken(gc models.GlobalConfig) error {
	// Send request
	reqUrl := fmt.Sprintf("%s/oauth/token", constants.Auth0DomainDev)
	reqData := url.Values{}
	reqData.Set("grant_type", "authorization_code")
	reqData.Set("client_id", constants.Auth0ClientIDDev)
	reqData.Set("refresh_token", gc.Auth.RefreshToken)
	res, err := http.Post(
		reqUrl,
		"application/x-www-form-urlencoded",
		strings.NewReader(reqData.Encode()),
	)
	if err != nil {
		console.Verbose("Error while refreshing access token: %s", err)
		console.ErrorPrint(constants.ErrMsgAuthFailed)
		os.Exit(1)
	}

	if res.StatusCode != 200 {
		console.Verbose("Received non-200 status while refreshing access token: %s", res.Status)
		return console.Error(constants.ErrMsgAuthFailed)
	}

	// Parse response
	authConfig, err := ParseAccessTokenResponse(res)
	if err != nil {
		console.Verbose("Error while parsing access token response: %s", err)
		console.ErrorPrint(constants.ErrMsgAuthFailed)
		os.Exit(1)
	}

	// Save global config
	gc.Auth = authConfig
	err = configio.SaveGlobalConfig(gc)
	if err != nil {
		console.Verbose("Error while saving global config: %s", err)
		return console.Error(constants.ErrMsgAuthFailed)
	}

	return nil
}
