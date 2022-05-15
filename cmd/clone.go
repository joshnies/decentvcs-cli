package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/models"
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
	clonePath := c.Args().Get(1)
	if clonePath == "" {
		clonePath = "."
	}
	clonePath, err := filepath.Abs(clonePath)
	if err != nil {
		return console.Error("Invalid project path")
	}

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

	// Get default branch
	apiUrl = api.BuildURLf("projects/%s/branches/default?join_commit=true", project.ID)
	res, err = httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse default branch
	var branch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}

	console.Verbose("Found default branch with commit index %d", branch.Commit.Index)

	if len(maps.Values(branch.Commit.HashMap)) == 0 {
		return console.Error("No committed files found for branch \"%s\"", branch.Name)
	}

	console.Info("Cloning project \"%s\" into \"%s\"...", project.Name, clonePath)

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
	console.Verbose("Downloading files...")

	if config.I.Verbose {
		for _, v := range maps.Values(branch.Commit.HashMap) {
			console.Verbose(" - %s", v)
		}
	}

	dataMap, err := storj.DownloadBulk(projectConfig.ProjectID, maps.Values(branch.Commit.HashMap))
	if err != nil {
		return err
	}

	// Iterate over downloaded files and write each to disk
	for _, hash := range maps.Keys(dataMap) {
		// Get file path
		var path string
		for p, h := range branch.Commit.HashMap {
			if hash == h {
				path = p
				break
			}
		}

		if path == "" {
			return console.Error("Failed to download file with hash %s", hash)
		}

		// Write file to local filesystem
		err = ioutil.WriteFile(path, dataMap[hash], 0644)
		if err != nil {
			return console.Error("Failed to write file (%s) after downloading: %s", path, err)
		}
	}

	return nil
}
