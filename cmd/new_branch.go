package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
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
	regex := regexp.MustCompile(`^[\w\-\.]+$`)
	if !regex.MatchString(branchName) {
		return console.Error("Invalid branch name; must be alphanumeric, and can contain dashes or periods")
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
	projectConfig.CurrentBranchName = branch.Name
	projectConfigPath, err := vcs.GetProjectConfigPath()
	if err != nil {
		return err
	}

	if _, err = vcs.SaveProjectConfig(filepath.Dir(projectConfigPath), projectConfig); err != nil {
		return err
	}

	console.Info("Created and switched to branch %s", branch.Name)
	return nil
}
