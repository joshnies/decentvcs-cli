package vcs

import (
	"fmt"
	"net/http"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/models"
)

// Delete all commits ahead of the given index for the specified branch.
// All unique file uploads are immediately deleted (based on commit hash maps).
func DeleteCommitsAheadOfIndex(projectConfig models.ProjectConfig, branchID string, index int) error {
	// Delete all commits ahead of the given index for the specified branch
	console.Verbose("Deleting commits ahead of commit #%d...", index)
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s/commits?after=%d", config.I.VCS.ServerHost, projectConfig.ProjectSlug, branchID, index)
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
	reqUrl = fmt.Sprintf("%s/projects/%s/storage/unused", config.I.VCS.ServerHost, projectConfig.ProjectSlug)
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
