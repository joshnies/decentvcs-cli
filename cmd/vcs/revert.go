package vcs

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/commits"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/corefs"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/models"
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
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/commits/index/%d", config.I.VCS.ServerHost, projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
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

	// Parse commit
	var currentCommit models.Commit
	err = json.NewDecoder(res.Body).Decode(&currentCommit)
	if err != nil {
		return console.Error("Failed to parse commit: %s", err)
	}

	// Reset all changes to current commit
	err = corefs.ResetChanges(gc, !c.Bool("no-confirm"))
	if err != nil {
		console.ErrorPrint("An error occurred while resetting changes")
		return err
	}

	// Sync to last commit
	return commits.SyncToCommit(gc, projectConfig, currentCommit.Index-1, !c.Bool("no-confirm"))
}
