package vcscmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
)

// Create a new branch.
func NewBranch(c *cli.Context) error {
	auth.HasToken()

	// Get branch name from args
	branchName := c.Args().First()
	if branchName == "" {
		return console.Error("Branch name is required")
	}

	// Validate branch name
	regex := regexp.MustCompile(`^[\w\-]+$`)
	if !regex.MatchString(branchName) {
		return console.Error("Invalid branch name; must be alphanumeric with dashes")
	}

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Create branch
	bodyJson, err := json.Marshal(models.BranchCreateDTO{
		Name:        branchName,
		CommitIndex: projectConfig.CurrentCommitIndex,
	})
	if err != nil {
		return err
	}

	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches", config.I.VCS.ServerHost, projectConfig.ProjectSlug)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var branch models.Branch
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}

	// Set current branch
	projectConfig.CurrentBranchName = branch.ID
	_, err = vcs.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	console.Info("Created and switched to branch %s", branch.Name)
	return nil
}
