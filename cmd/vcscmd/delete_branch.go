package vcscmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
)

// Soft-delete the specified branch.
// Does NOT effect the branch's commits.
func DeleteBranch(c *cli.Context) error {
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
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.VCS.ServerHost, projectConfig.ProjectID, branchName)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var branch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}

	// Prevent deletion of current branch
	if branch.ID == projectConfig.CurrentBranchID {
		return console.Error("You cannot delete the current branch. Please switch to another branch first.")
	}

	// Ask for confirmation
	if !c.Bool("no-confirm") {
		console.Warning("Are you sure you want to delete the branch \"%s\"?", branchName)
		var answer string
		fmt.Scanln(&answer)
	}

	// Soft-delete branch
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectID, branchName)
	req, err = http.NewRequest("DELETE", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	res.Body.Close()
	console.Success("Branch deleted")
	return nil
}
