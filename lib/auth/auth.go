package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

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
	var resData map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		console.Verbose("Error while parsing access token refresh response: %s", err)
		return console.Error(constants.ErrMsgAuthFailed)
	}

	accessToken := resData["access_token"]
	if accessToken == nil {
		console.Verbose("Access token not found in response")
		return console.Error(constants.ErrMsgAuthFailed)
	}

	refreshToken := resData["refresh_token"]
	if refreshToken == nil {
		console.Verbose("Refresh token not found in response")
		return console.Error(constants.ErrMsgAuthFailed)
	}

	idToken := resData["id_token"]
	if idToken == nil {
		console.Verbose("ID token not found in response")
		return console.Error(constants.ErrMsgAuthFailed)
	}

	expiresInRaw := resData["expires_in"]
	if expiresInRaw == nil {
		console.Verbose("Expires in not found in response")
		return console.Error(constants.ErrMsgAuthFailed)
	}
	expiresIn := int(expiresInRaw.(float64))

	// Update global config
	gc.Auth.AccessToken = accessToken.(string)
	gc.Auth.RefreshToken = refreshToken.(string)
	gc.Auth.IDToken = idToken.(string)
	gc.Auth.ExpiresIn = expiresIn

	// Save global config
	err = configio.SaveGlobalConfig(gc)
	if err != nil {
		console.Verbose("Error while saving global config: %s", err)
		return console.Error(constants.ErrMsgAuthFailed)
	}

	return nil
}
