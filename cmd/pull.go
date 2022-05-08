package cmd

import (
	"encoding/json"
	"os"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get newer commits from remote for current branch
	apiUrl := api.BuildURLf("projects/%s/branches/%s/commits?after=%s", projectConfig.ProjectID, projectConfig.CurrentBranchID, projectConfig.CurrentCommitID)
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		console.Verbose("Error fetching commits: %s", err)
		return console.Error("Failed to fetch commits")
	}

	// Parse response
	var commits []models.Commit
	err = json.NewDecoder(res.Body).Decode(&commits)
	if err != nil {
		console.Verbose("Error parsing commits from API response: %s", err)
		return console.Error("Failed to fetch commits")
	}

	// Return if no new commits found
	if len(commits) == 0 {
		console.Info("No changes to pull.")
		return nil
	}

	for _, commit := range commits {
		// Download snapshots
		dataMap, err := storj.DownloadBulk(commit.ProjectID, commit.ID, commit.SnapshotPaths)
		if err != nil {
			console.Verbose("Error downloading files: %s", err)
			return console.Error("Failed to download files from storage")
		}

		// Create new files in local file system
		for path, data := range dataMap {
			file, err := os.Open(path)
			if err != nil {
				console.Verbose("Error opening file: %s", err)
				return console.Error("Failed to open file")
			}
			defer file.Close()

			_, err = file.Write(data)
			if err != nil {
				console.Verbose("Error writing file: %s", err)
				return console.Error("Failed to write file")
			}
		}

		// TODO: Download patches
		// TODO: Apply patches
		// TODO: Delete deleted files
	}

	console.Success("Successful")
	return nil
}
