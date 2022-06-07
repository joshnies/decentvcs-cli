package vcs

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/corefs"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
)

// Print list of current changes
func GetChanges(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ current commit
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, projectConfig.CurrentBranchID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Detect local changes
	// TODO: Use user-provided project path if available
	fc, err := corefs.DetectFileChanges(".", currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	// If there are no changes, exit
	changeCount := len(fc.CreatedFilePaths) + len(fc.ModifiedFilePaths) + len(fc.DeletedFilePaths)
	if changeCount == 0 {
		console.Info("No changes detected")
		return nil
	}

	return nil
}
