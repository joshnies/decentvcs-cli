package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/TwiN/go-color"
	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
)

// Switch to the specified branch.
// This will also sync to the latest commit on that branch.
func UseBranch(c *cli.Context) error {
	auth.HasToken()

	// Get the branch name
	branchName := c.Args().First()
	if branchName == "" {
		return cli.Exit("You must specify a branch name", 1)
	}

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get specified branch
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.VCS.ServerHost, projectConfig.ProjectSlug, branchName)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Parse response
	var branch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}
	res.Body.Close()

	// Set the current branch in project config
	projectConfig.CurrentBranchName = branch.Name
	projectConfigPath, err := vcs.GetProjectConfigPath()
	if err != nil {
		return err
	}

	if _, err = vcs.SaveProjectConfig(filepath.Dir(projectConfigPath), projectConfig); err != nil {
		return err
	}

	// Reset local changes if specified branch points to a different commit than current
	if projectConfig.CurrentCommitIndex != branch.Commit.Index {
		// Reset local changes
		err = vcs.ResetChanges(!c.Bool("yes"))
		if err != nil {
			return err
		}
	}

	// Sync
	if projectConfig.CurrentCommitIndex != branch.Commit.Index {
		err = vcs.SyncToCommit(projectConfig, branch.Commit.Index, true)
		if err != nil {
			return err
		}
	}

	console.Info("Switched to branch %s", color.InBold(branchName))
	return nil
}
