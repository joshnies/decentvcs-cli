package cmd

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Clone remote project at default branch to local machine.
// TODO: Clone projects by name
// TODO: Scope projects to a team or user
func CloneProject(c *cli.Context) error {
	gc := auth.Validate()

	// Get project ID
	projectID := c.Args().First()
	if projectID == "" {
		return console.Error("Please provide a project ID")
	}

	// Check if already in project directory
	if _, err := os.Stat(constants.ProjectFileName); errors.Is(err, os.ErrNotExist) {
		return console.Error("Project already exists in current directory")
	}

	// Get project
	apiUrl := api.BuildURLf("projects/%s")
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse project
	var project models.Project
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return err
	}

	// Get default branch
	apiUrl = api.BuildURLf("projects/%s/branches/default")
	res, err = httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse default branch
	var branch models.Branch
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}

	console.Info("Cloning project...")

	// TODO: Create project config file
	// TODO: Download all files

	return nil
}
