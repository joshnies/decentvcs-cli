package cmd

import (
	"bytes"
	"encoding/json"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
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
	apiUrl := api.BuildURLf("projects/%s/branches/%s", projectConfig.ProjectID, branchName)
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		console.ErrorPrint("Error getting branch:")
		return err
	}

	// Parse branch
	branch := models.Branch{}
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		console.Verbose("Failed to parse branch \"%s\": %s", branchName, err)
		return console.Error(constants.ErrMsgInternal)
	}

	// Update project with default branch
	apiUrl = api.BuildURLf("projects/%s", projectConfig.ProjectID)
	bodyData := models.Project{
		DefaultBranchID: branch.ID,
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		console.Verbose("Failed to convert project DTO to JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}
	body := bytes.NewBuffer(bodyJson)
	_, err = httpw.Post(apiUrl, body, gc.Auth.AccessToken)
	if err != nil {
		console.ErrorPrint("Error setting default branch:")
		return err
	}

	console.Info("Default branch set to \"%s\"", branchName)
	return nil
}