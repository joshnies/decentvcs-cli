package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/constants"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

// Set default branch for project.
func SetDefaultBranch(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get branch name from args
	branchName := c.Args().Get(0)

	// Get branch
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s", config.I.API.Host, projectConfig.ProjectID, branchName)
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

	// Parse branch
	branch := models.Branch{}
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		console.Verbose("Failed to parse branch \"%s\": %s", branchName, err)
		return console.Error(constants.ErrMsgInternal)
	}

	// Update project with default branch
	bodyData := models.Project{
		DefaultBranchID: branch.ID,
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		console.Verbose("Failed to convert project DTO to JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}
	reqUrl = fmt.Sprintf("%s/projects/%s", config.I.API.Host, projectConfig.ProjectID)
	req, err = http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	res.Body.Close()
	console.Info("Default branch set to \"%s\"", branchName)
	return nil
}
