package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/commits"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/lib/projects"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

// Switch to the specified branch.
// This will also sync to the latest commit on that branch.
func UseBranch(c *cli.Context) error {
	gc := auth.Validate()

	// Get the branch name
	branchName := c.Args().First()
	if branchName == "" {
		return cli.Exit("You must specify a branch name", 1)
	}

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get specified branch
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, branchName)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
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
	projectConfig.CurrentBranchID = branch.ID
	projectConfig, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	// Reset local changes if specified branch points to a different commit than current
	if projectConfig.CurrentCommitIndex != branch.Commit.Index {
		// Reset local changes
		err = projects.ResetChanges(gc, !c.Bool("no-confirm"))
		if err != nil {
			return err
		}
	}

	// Sync
	err = commits.SyncToCommit(gc, projectConfig, branch.Commit.Index, true)
	if err != nil {
		return err
	}

	return nil
}
