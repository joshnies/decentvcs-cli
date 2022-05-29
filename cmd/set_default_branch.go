package cmd

import (
	"bytes"
	"encoding/json"

	"github.com/joshnies/quanta-cli/config"
	"github.com/joshnies/quanta-cli/constants"
	"github.com/joshnies/quanta-cli/lib/api"
	"github.com/joshnies/quanta-cli/lib/auth"
	"github.com/joshnies/quanta-cli/lib/console"
	"github.com/joshnies/quanta-cli/lib/httpw"
	"github.com/joshnies/quanta-cli/models"
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
	bodyData := models.Project{
		DefaultBranchID: branch.ID,
	}
	bodyJson, err := json.Marshal(bodyData)
	if err != nil {
		console.Verbose("Failed to convert project DTO to JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}
	_, err = httpw.Post(httpw.RequestParams{
		URL:         api.BuildURLf("projects/%s", projectConfig.ProjectID),
		Body:        bytes.NewBuffer(bodyJson),
		AccessToken: gc.Auth.AccessToken,
	})
	if err != nil {
		console.ErrorPrint("Error setting default branch:")
		return err
	}

	console.Info("Default branch set to \"%s\"", branchName)
	return nil
}
