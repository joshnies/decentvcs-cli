package cmd

import (
	"encoding/json"

	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/projects"
	"github.com/joshnies/qc/models"
	"github.com/urfave/cli/v2"
)

// Print list of current changes
func GetChanges(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ current commit
	apiUrl := api.BuildURLf("projects/%s/branches/%s?join_commit=true", projectConfig.ProjectID, projectConfig.CurrentBranchID)
	currentBranchRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}
	defer currentBranchRes.Body.Close()

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(currentBranchRes.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Detect local changes
	// TODO: Use user-provided project path if available
	fc, err := projects.DetectFileChanges(".", currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	// If there are no changes, exit
	changeCount := len(fc.CreatedFilePaths) + len(fc.ModifiedFilePaths) + len(fc.DeletedFilePaths)
	if changeCount == 0 {
		console.Info("No changes detected")
		return nil
	}

	return nil
}
