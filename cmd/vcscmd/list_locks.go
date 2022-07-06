package vcscmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
)

// List current file locks on the current or specified branch.
func ListLocks(c *cli.Context) error {
	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get name or ID of branch to list locks for
	var branch models.Branch
	branchNameOrID := c.String("branch")
	if branchNameOrID == "" {
		// Default to current branch
		branchNameOrID = projectConfig.CurrentBranchID
	}

	// Get branch
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectID, branchNameOrID)
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Parse response
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}

	if len(branch.Locks) == 0 {
		console.Warning("No files locked in branch %s", branch.Name)
		return nil
	}

	// Print locks
	// TODO: Print "locked by" user names instead of Stytch user IDs
	console.Info("Locks for branch %s:", branch.Name)
	for path, lockedByUserID := range branch.Locks {
		console.Info("  - %s: %s", path, lockedByUserID)
	}

	return nil
}
