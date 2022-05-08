package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/models"
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
	apiUrl := api.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID)
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
	fcdRes, err := projects.DetectFileChanges(currentBranch.Commit.State)
	if err != nil {
		return err
	}

	// If there are no changes, exit
	if len(fcdRes.Changes) == 0 {
		console.Info("No changes detected")
		return nil
	}

	// Print changes
	console.Info("%d changes found:", len(fcdRes.Changes))

	for _, change := range fcdRes.Changes {
		switch change.Type {
		case models.FileWasCreated:
			fmt.Printf(color.Ize(color.Green, "  + %s\n"), change.Path)
		case models.FileWasModified:
			// TODO: Print lines added and removed
			fmt.Printf(color.Ize(color.Cyan, "  * %s\n"), change.Path)
		case models.FileWasDeleted:
			fmt.Printf(color.Ize(color.Red, "  - %s\n"), change.Path)
		}
	}

	return nil
}
