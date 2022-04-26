package lib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/models"
)

// Get Storj access grant for project.
// Gives read-write access to project folder in Storj bucket.
func GetAccessGrant() (string, error) {
	// Use existing access grant if it exists and has not expired
	projectConfig, err := config.GetProjectConfig()
	if err == nil {
		expiration := time.Unix(projectConfig.AccessGrantExpiration, 0)

		if time.Now().Before(expiration) {
			return projectConfig.AccessGrant, nil
		}
	} else {
		fmt.Println("Error getting project config")
		return "", err
	}

	projectId := projectConfig.ProjectID

	// Get new access grant from API
	res, err := http.Get(BuildURL(fmt.Sprintf("projects/%s/access_grant", projectId)))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("failed to get access grant from API (status: %s)", res.Status)
	}

	// Parse response
	var decodedRes models.AccessGrantResponse
	err = json.NewDecoder(res.Body).Decode(&decodedRes)
	if err != nil {
		fmt.Println("Error parsing API response")
		return "", err
	}

	accessGrant := decodedRes.AccessGrant

	// Write access grant to project config
	projectConfig, err = WriteProjectConfig(".", models.ProjectFileData{
		AccessGrant: accessGrant,
		// TODO: Enforce this 24-hour expiration in Storj itself
		AccessGrantExpiration: time.Now().Add(time.Hour * 24).Unix(),
	})
	if err != nil {
		fmt.Println("Error writing project config")
		return "", err
	}

	return accessGrant, nil
}
