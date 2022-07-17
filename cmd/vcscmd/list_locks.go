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
	"github.com/samber/lo"
	"github.com/stytchauth/stytch-go/v5/stytch"
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
		branchNameOrID = projectConfig.CurrentBranchName
	}

	// Get branch
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectSlug, branchNameOrID)
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
		console.Info("No files locked in branch %s", branch.Name)
		return nil
	}

	// Get "locked by" users from Stytch to get their full names
	lockedByUserIDs := lo.Values(branch.Locks)
	lockedByUserNames := make(map[string]string)
	for _, userID := range lockedByUserIDs {
		// Get Stytch user from server
		reqUrl = fmt.Sprintf("%s/stytch/users/%s", config.I.VCS.ServerHost, userID)
		req, _ = http.NewRequest("GET", reqUrl, nil)
		req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
		res, err = httpClient.Do(req)
		if err != nil {
			return err
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			return err
		}

		// Parse response
		var stytchUser stytch.UsersGetResponse
		err = json.NewDecoder(res.Body).Decode(&stytchUser)
		if err != nil {
			return err
		}

		if stytchUser.Name.FirstName != "" || stytchUser.Name.LastName != "" {
			// Use name
			lockedByUserNames[userID] = stytchUser.Name.FirstName + " " + stytchUser.Name.LastName
		} else {
			// Use first email
			lockedByUserNames[userID] = stytchUser.Emails[0].Email
		}
	}

	// Print locks
	console.Info("Locks for branch %s:", branch.Name)
	for path, lockedByUserID := range branch.Locks {
		console.Info("  - %s: %s", path, lockedByUserNames[lockedByUserID])
	}

	return nil
}
