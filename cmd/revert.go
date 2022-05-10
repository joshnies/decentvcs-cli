package cmd

import (
	"encoding/json"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/commits"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Reset all changes and sync to last commit.
func Revert(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	if projectConfig.CurrentCommitIndex <= 0 {
		return console.Error("Current commit index is invalid. Please check your project config file.")
	}

	// Get current commit by index
	apiUrl := api.BuildURLf("projects/%s/commits/index/%d", projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
	commitRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse commit
	var currentCommit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&currentCommit)
	if err != nil {
		return console.Error(constants.ErrMsgInternal)
	}

	// Reset all changes to current commit
	err = projects.ResetChanges(c)
	if err != nil {
		return err
	}

	// Sync to last commit
	return commits.SyncToCommit(gc, projectConfig, currentCommit.Index-1)
}
