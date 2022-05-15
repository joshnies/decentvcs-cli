package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/models"
	"github.com/urfave/cli/v2"
)

// Print info for the current project, branch, and commit
func PrintStatus(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get project
	apiUrl := api.BuildURLf("projects/%s", projectConfig.ProjectID)
	projectRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return console.Error("Failed to get project: %s", err)
	}

	// Parse project
	var project models.Project
	err = json.NewDecoder(projectRes.Body).Decode(&project)
	if err != nil {
		return console.Error("Failed to parse project: %s", err)
	}

	// Get branch
	apiUrl = api.BuildURLf("projects/%s/branches/%s", projectConfig.ProjectID, projectConfig.CurrentBranchID)
	branchRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return console.Error("Failed to get branch: %s", err)
	}

	// Parse branch
	var branch models.Branch
	err = json.NewDecoder(branchRes.Body).Decode(&branch)
	if err != nil {
		return console.Error("Failed to parse branch: %s", err)
	}

	// Get commit
	apiUrl = api.BuildURLf("projects/%s/commits/index/%d", projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
	commitRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return console.Error("Failed to get commit: %s", err)
	}

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&commit)
	if err != nil {
		return console.Error("Failed to parse commit: %s", err)
	}

	fmt.Printf(color.Ize(color.Cyan, "Project: ")+"%s (%s)\n", project.Name, project.ID)
	fmt.Printf(color.Ize(color.Cyan, "Branch:  ")+"%s (%s)\n", branch.Name, branch.ID)
	fmt.Printf(color.Ize(color.Cyan, "Commit:  ")+"#%d (%s)\n", commit.Index, commit.ID)

	return nil
}
