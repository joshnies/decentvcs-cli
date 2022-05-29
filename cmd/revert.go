package cmd

import (
	"encoding/json"

	"github.com/joshnies/quanta-cli/config"
	"github.com/joshnies/quanta-cli/lib/api"
	"github.com/joshnies/quanta-cli/lib/auth"
	"github.com/joshnies/quanta-cli/lib/commits"
	"github.com/joshnies/quanta-cli/lib/console"
	"github.com/joshnies/quanta-cli/lib/httpw"
	"github.com/joshnies/quanta-cli/lib/projects"
	"github.com/joshnies/quanta-cli/models"
	"github.com/urfave/cli/v2"
)

// Reset all local changes and sync to last commit.
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
		return console.Error("Failed to parse commit: %s", err)
	}

	// Reset all changes to current commit
	err = projects.ResetChanges(gc, !c.Bool("no-confirm"))
	if err != nil {
		console.ErrorPrint("An error occurred while resetting changes")
		return err
	}

	// Sync to last commit
	return commits.SyncToCommit(gc, projectConfig, currentCommit.Index-1, !c.Bool("no-confirm"))
}
