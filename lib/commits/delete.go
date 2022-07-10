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
func DeleteCommitsAheadOfIndex(projectConfig models.ProjectConfig, branchIDOrName string, index int) error {
	// Get file hashes for all files that only appear in commits ahead of the given index
	console.Verbose("Getting all unique file hashes ahead of commit #%d...", index)
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s/unique_hashes?after_commit_index=%d", config.I.VCS.ServerHost, projectConfig.ProjectID, branchIDOrName, index)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to get unique file hashes: %v", err)
	}
	if err := httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Delete all commits ahead of the given index for the specified branch
	console.Verbose("Deleting commits ahead of commit #%d...", index)
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s/commits?after=%d", config.I.VCS.ServerHost, projectConfig.ProjectID, branchIDOrName, index)
	req, err = http.NewRequest("DELETE", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err = httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to delete commits: %v", err)
	}
	if err := httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// TODO: Delete all unique file uploads from storage

	return nil
}
