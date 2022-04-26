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
		return "", err
	}

	projectId := projectConfig.ProjectID

	// Get new access grant from API
	res, err := http.Get(BuildURL(fmt.Sprintf("%s/access_grant", projectId)))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Parse response
	var decodedRes models.AccessGrantResponse
	err = json.NewDecoder(res.Body).Decode(&decodedRes)
	if err != nil {
		return "", err
	}

	accessGrant := decodedRes.AccessGrant

	// Write access grant to project file
	err = WriteProjectFile(projectId, models.ProjectFileData{
		AccessGrant: accessGrant,
	})
	if err != nil {
		return "", err
	}

	return accessGrant, nil
}
