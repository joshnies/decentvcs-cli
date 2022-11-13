package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/models"
)

// Logs a fatal error if the user not not have an existing auth token for DecentVCS.
func HasToken() {
	// Check if config has auth data
	if config.I.Auth.SessionToken == "" {
		log.Fatal("not authenticated, please run `dvcs login`")
	}
}

// Create a new access key.
func CreateAccessKey(teamName string, scope string) models.AccessKey {
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/teams/%s/access_keys", config.I.VCS.ServerHost, teamName)
	req, _ := http.NewRequest("POST", reqUrl, nil)
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if err := httpvalidation.ValidateResponse(res); err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	var accessKey models.AccessKey
	err = json.NewDecoder(res.Body).Decode(&accessKey)
	if err != nil {
		log.Fatal(err)
	}

	return accessKey
}
