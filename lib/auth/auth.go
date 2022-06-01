package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/configio"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/models"
)

// Returns true if the user is logged in, false otherwise.
func Validate() models.GlobalConfig {
	// Get global config
	gc, err := config.GetGlobalConfig()
	if err != nil {
		console.Verbose("Error while getting global config: %s", err)
		console.Fatal(constants.ErrMsgAuthFailed)
	}

	// Get or refresh access token
	gc, err = UseAccessToken(gc)
	if err != nil {
		console.Verbose("Error while getting or refreshing access token: %s", err)
		console.Fatal(constants.ErrMsgAuthFailed)
	}

	return gc
}

// Get or refresh access token
func UseAccessToken(gc models.GlobalConfig) (models.GlobalConfig, error) {
	// If access token has not yet expired, return it
	if gc.Auth.AuthenticatedAt+gc.Auth.ExpiresIn > time.Now().Unix() {
		return gc, nil
	}

	// Refresh access token
	console.Verbose("Access token has expired, refreshing...")
	gc, err := refreshAccessToken(gc)
	if err != nil {
		return models.GlobalConfig{}, err
	}

	console.Verbose("Access token refreshed")

	return gc, nil
}

// Parse access token response from Auth0 Authentication API
func ParseAccessTokenResponse(res *http.Response) (models.GlobalConfigAuth, error) {
	// Parse response
	var authConfig models.GlobalConfigAuth
	err := json.NewDecoder(res.Body).Decode(&authConfig)
	if err != nil {
		return models.GlobalConfigAuth{}, err
	}

	// Validate response
	// NOTE: Does not check if refresh token was returned, since it's not returned with all
	// grant types.
	if authConfig.AccessToken == "" {
		return models.GlobalConfigAuth{}, console.Error("Access token not found in response")
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
func refreshAccessToken(gc models.GlobalConfig) (models.GlobalConfig, error) {
	// Send request
	reqUrl := fmt.Sprintf("%s/oauth/token", constants.Auth0DomainDev)
	reqData := url.Values{}
	reqData.Set("grant_type", "refresh_token")
	reqData.Set("client_id", constants.Auth0ClientIDDev)
	reqData.Set("refresh_token", gc.Auth.RefreshToken)
	res, err := http.Post(
		reqUrl,
		"application/x-www-form-urlencoded",
		strings.NewReader(reqData.Encode()),
	)
	if err != nil {
		console.Verbose("Error while refreshing access token: %s", err)
		return models.GlobalConfig{}, console.Error(constants.ErrMsgAuthFailed)
	}

	if res.StatusCode != 200 {
		console.Verbose("Received non-200 status while refreshing access token: %s", res.Status)
		return models.GlobalConfig{}, console.Error(constants.ErrMsgAuthFailed)
	}

	// Parse response
	authConfig, err := ParseAccessTokenResponse(res)
	if err != nil {
		console.Verbose("Error while parsing access token response: %s", err)
		return models.GlobalConfig{}, console.Error(constants.ErrMsgAuthFailed)
	}

	// Retain refresh token
	authConfig.RefreshToken = gc.Auth.RefreshToken

	// Save global config
	gc.Auth = authConfig
	err = configio.SaveGlobalConfig(gc)
	if err != nil {
		console.Verbose("Error while saving global config: %s", err)
		return models.GlobalConfig{}, console.Error(constants.ErrMsgAuthFailed)
	}

	return gc, nil
}
