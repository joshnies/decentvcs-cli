package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
)

// Reset all local changes and sync to last commit.
func Revert(c *cli.Context) error {
	auth.HasToken()

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	if projectConfig.CurrentCommitIndex <= 0 {
		return console.Error("Current commit index is invalid. Please check your project config file.")
	}

	// Get current commit by index
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/commits/%d", config.I.VCS.ServerHost, projectConfig.ProjectSlug, projectConfig.CurrentCommitIndex)
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

	// Parse commit
	var currentCommit models.Commit
	err = json.NewDecoder(res.Body).Decode(&currentCommit)
	if err != nil {
		return console.Error("Failed to parse commit: %s", err)
	}

	// Reset all changes to current commit
	err = vcs.ResetChanges(!c.Bool("yes"))
	if err != nil {
		console.ErrorPrint("An error occurred while resetting changes")
		return err
	}

	// Sync to last commit
	return vcs.SyncToCommit(projectConfig, currentCommit.Index-1, !c.Bool("yes"))
}
