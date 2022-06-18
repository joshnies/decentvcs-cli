package auth0

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"gopkg.in/yaml.v3"
)

// Parse access token response from Auth0 Authentication API
func ParseAccessTokenResponse(res *http.Response) (config.AuthConfig, error) {
	// Parse response
	var authConfig config.AuthConfig
	err := json.NewDecoder(res.Body).Decode(&authConfig)
	if err != nil {
		return config.AuthConfig{}, err
	}

	// Validate response
	// NOTE: Does not check if refresh token was returned, since it's not returned with all
	// grant types.
	if authConfig.AccessToken == "" {
		return config.AuthConfig{}, errors.New("\"access_token\" not found in response")
	}

	if authConfig.IDToken == "" {
		return config.AuthConfig{}, errors.New("\"id_token\" not found in response")
	}

	if authConfig.ExpiresIn == 0 {
		return config.AuthConfig{}, errors.New("\"expires_in\" not found in response")
	}

	// Add additional data
	authConfig.AuthenticatedAt = time.Now().Unix()
	return authConfig, nil
}

// Refresh access token
func RefreshAccessToken() {
	// Send request
	reqUrl := fmt.Sprintf("%s/oauth/token", constants.Auth0DomainDev)
	reqData := url.Values{}
	reqData.Set("grant_type", "refresh_token")
	reqData.Set("client_id", constants.Auth0ClientIDDev)
	reqData.Set("refresh_token", config.I.Auth.RefreshToken)
	res, err := http.Post(
		reqUrl,
		"application/x-www-form-urlencoded",
		strings.NewReader(reqData.Encode()),
	)
	if err != nil {
		log.Fatalf("failed to refresh access token with Auth0: %v", err)
	}

	if res.StatusCode != 200 {
		log.Fatalf("received bad status code from Auth0 while refreshing access token: %s", res.Status)
	}

	// Parse response
	authConfig, err := ParseAccessTokenResponse(res)
	if err != nil {
		log.Fatalf("failed to parse access token response from Auth0: %v", err)
	}

	// Retain refresh token if it exists
	if config.I.Auth.RefreshToken != "" {
		authConfig.RefreshToken = config.I.Auth.RefreshToken
	}

	// Write config to file
	config.I.Auth = authConfig
	cYaml, err := yaml.Marshal(config.I)
	if err != nil {
		log.Fatalf("error while converting config data to yaml: %v", err)
	}

	err = ioutil.WriteFile(config.GetConfigPath(), cYaml, 0644)
	if err != nil {
		log.Fatalf("error while writing config: %v", err)
	}
}
