package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/joshnies/quanta-cli/config"
	"github.com/joshnies/quanta-cli/constants"
	"github.com/joshnies/quanta-cli/lib/api"
	"github.com/joshnies/quanta-cli/lib/auth"
	"github.com/joshnies/quanta-cli/lib/console"
	"github.com/joshnies/quanta-cli/lib/httpw"
	"github.com/joshnies/quanta-cli/lib/storage"
	"github.com/joshnies/quanta-cli/models"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

// Clone remote project at default branch to local machine.
func CloneProject(c *cli.Context) error {
	gc := auth.Validate()

	// Get project blob from first arg
	projectBlob := c.Args().First()
	if projectBlob == "" {
		return console.Error("Please specify a project in the format of \"<author_or_team_alias>/<project_name>\"")
	}

	// Get clone path from second arg
	clonePath, err := filepath.Abs(c.String("path"))
	if err != nil {
		return console.Error("Invalid path")
	}

	// Get branch name
	branchName := c.String("branch")

	// Check if already in project directory
	if _, err := os.Stat(filepath.Join(clonePath, constants.ProjectFileName)); !os.IsNotExist(err) {
		return console.Error("A project already exists in the current directory")
	}

	// Get project
	apiUrl := api.BuildURLf("projects/blob/%s", projectBlob)
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

	var branch models.BranchWithCommit
	if branchName == "" {
		// Get default branch
		apiUrl = api.BuildURLf("projects/%s/branches/default?join_commit=true", project.ID)
		res, err = httpw.Get(apiUrl, gc.Auth.AccessToken)
		if err != nil {
			return err
		}

		// Parse branch
		err = json.NewDecoder(res.Body).Decode(&branch)
		if err != nil {
			return err
		}
	} else {
		// Get specified branch
		apiUrl = api.BuildURLf("projects/%s/branches/%s?join_commit=true", project.ID, branchName)
		res, err = httpw.Get(apiUrl, gc.Auth.AccessToken)
		if err != nil {
			return err
		}

		// Parse branch
		err = json.NewDecoder(res.Body).Decode(&branch)
		if err != nil {
			return err
		}
	}

	if len(maps.Values(branch.Commit.HashMap)) == 0 {
		return console.Error("No committed files found for branch \"%s\"", branch.Name)
	}

	// Create clone directory resursively
	err = os.MkdirAll(clonePath, 0755)
	if err != nil {
		return err
	}

	console.Info("Cloning project \"%s\" with branch \"%s\" into \"%s\"...", project.Name, branch.Name, clonePath)
	console.Verbose("Branch commit index: %d", branch.Commit.Index)

	// Create project config file
	console.Verbose("Creating project config file...")
	projectConfig := models.ProjectConfig{
		ProjectID:          project.ID,
		CurrentBranchID:    branch.ID,
		CurrentCommitIndex: branch.Commit.Index,
	}

	projectConfig, err = config.SaveProjectConfig(clonePath, projectConfig)
	if err != nil {
		return err
	}

	console.Verbose("Project config file created")

	// Download all files
	err = storage.DownloadMany(projectConfig.ProjectID, clonePath, branch.Commit.HashMap)
	if err != nil {
		return err
	}

	return nil
}
