package cmd

import (
	"encoding/json"
	"fmt"
	"time"

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
	startTime := time.Now()
	fc, err := projects.DetectFileChanges(currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	timeElapsed := time.Since(startTime).Truncate(time.Microsecond)

	// If there are no changes, exit
	changeCount := len(fc.CreatedFilePaths) + len(fc.ModifiedFilePaths) + len(fc.DeletedFilePaths)
	if changeCount == 0 {
		console.Info("No changes detected (took %s)", timeElapsed)
		return nil
	}

	// Print changes
	console.Info("%d changes found:", changeCount)

	for _, path := range fc.CreatedFilePaths {
		fmt.Printf(color.Ize(color.Green, "  + %s\n"), path)
	}

	for _, path := range fc.ModifiedFilePaths {
		fmt.Printf(color.Ize(color.Blue, "  * %s\n"), path)
	}

	for _, path := range fc.DeletedFilePaths {
		fmt.Printf(color.Ize(color.Red, "  - %s\n"), path)
	}

	return nil
}
