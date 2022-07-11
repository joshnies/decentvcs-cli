package commits

import (
	"fmt"
	"net/http"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/models"
)

// Delete all commits ahead of the given index for the specified branch.
// All unique file uploads are immediately deleted (based on commit hash maps).
func DeleteCommitsAheadOfIndex(projectConfig models.ProjectConfig, branchID string, index int) error {
	// Delete all commits ahead of the given index for the specified branch
	console.Verbose("Deleting commits ahead of commit #%d...", index)
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s/commits?after=%d", config.I.VCS.ServerHost, projectConfig.ProjectID, branchID, index)
	req, _ := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to delete commits: %v", err)
	}
	if err := httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Delete all unused objects from storage
	console.Info("Deleting unused objects (this may take a while)...")
	reqUrl = fmt.Sprintf("%s/projects/%s/storage/unused", config.I.VCS.ServerHost, projectConfig.ProjectID)
	req, _ = http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err = httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to delete unused objects: %v", err)
	}
	if err := httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	return nil
}
