package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/dvcs/config"
	"github.com/joshnies/dvcs/constants"
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/console"
	"github.com/joshnies/dvcs/lib/httpvalidation"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/joshnies/dvcs/models"
	"github.com/urfave/cli/v2"
)

// Print list of current changes
func GetChanges(c *cli.Context) error {
	auth.HasToken()

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ current commit
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.VCS.ServerHost, projectConfig.ProjectSlug, projectConfig.CurrentBranchName)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
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
	fc, err := vcs.DetectFileChanges(currentBranch.Commit.HashMap)
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
