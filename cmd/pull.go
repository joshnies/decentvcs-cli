package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/gabstv/go-bsdiff/pkg/bspatch"
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

	// TODO: Show progress bar for:
	// - Downloading snapshots
	// - Writing snapshots to local fs
	// - Downloading patches
	// - Applying patches to local fs
	// - Removing deleted files from local fs
	for _, commit := range commits {
		// Download snapshots
		dataMap, err := storj.DownloadBulk(commit.ProjectID, commit.ID, commit.SnapshotPaths)
		if err != nil {
			console.Verbose("Error downloading snapshot files: %s", err)
			return console.Error("Failed to download new files from storage")
		}

		// Create new files in local fs
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

		// Download patches
		dataMap, err = storj.DownloadBulk(commit.ProjectID, commit.ID, commit.PatchPaths)
		if err != nil {
			console.Verbose("Error downloading patch files: %s", err)
			return console.Error("Failed to download modified files from storage")
		}

		// Apply patches to local fs
		for path, newData := range dataMap {
			// Open local file
			file, err := os.Open(path)
			if err != nil {
				console.Verbose("Error opening file: %s", err)
				return console.Error("Failed to open file: %s", path)
			}

			// Read local file as bytes
			oldData, err := ioutil.ReadAll(file)
			if err != nil {
				console.Verbose("Error reading file: %s", err)
				return console.Error("Failed to read file: %s", path)
			}

			patched, err := bspatch.Bytes(oldData, newData)
			if err != nil {
				console.Verbose("Error applying patch: %s", err)
				return console.Error("Failed to apply patch to file: %s", path)
			}

			// Write patched data to file
			_, err = file.Write(patched)
			if err != nil {
				console.Verbose("Error writing patched file: %s", err)
				return console.Error("Failed to write patched file: %s", path)
			}
		}

		// Remove deleted files from local fs
		for _, path := range commit.DeletedPaths {
			err := os.Remove(path)
			if err != nil {
				console.Verbose("Error deleting file: %s", err)
				return console.Error("Failed to delete file: %s", path)
			}
		}
	}

	console.Success("Successful")
	return nil
}
