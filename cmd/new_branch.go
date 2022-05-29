package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

// Create a new branch.
func NewBranch(c *cli.Context) error {
	gc := auth.Validate()

	// Get branch name from args
	branchName := c.Args().First()
	if branchName == "" {
		return console.Error("Branch name is required")
	}

	// Get project config
	projectConfig, err := config.GetProjectConfig()
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
	reqUrl := fmt.Sprintf("%s/projects/%s/branches", config.I.API.Host, projectConfig.ProjectID)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
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
	// TODO: Use user-specified project path
	projectConfig.CurrentBranchID = branch.ID
	_, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	console.Info("Created and switched to branch %s", branch.Name)
	return nil
}
