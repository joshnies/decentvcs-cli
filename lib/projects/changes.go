package projects

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/lib/util"
	"github.com/joshnies/qc-cli/models"
	"golang.org/x/exp/maps"
)

// Reset all local changes.
// This will:
//
// - Delete all created files
//
// - Revert all modified files to their original state
//
// - Recreate all deleted files
//
// @param gc Global config
// @param confirm Whether to prompt user for confirmation before resetting
//
func ResetChanges(gc models.GlobalConfig, confirm bool) error {
	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current commit
	apiUrl := api.BuildURLf("projects/%s/commits/index/%d", projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
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

	// Detect file changes
	fc, err := DetectFileChanges(commit.HashMap)
	if err != nil {
		return err
	}

	if len(fc.CreatedFilePaths) == 0 && len(fc.ModifiedFilePaths) == 0 && len(fc.DeletedFilePaths) == 0 {
		console.Info("No changes detected")
		return nil
	}

	// Prompt user for confirmation
	if confirm {
		console.Warning("You are about to reset all local changes. This will:")
		console.Warning("- Delete all created files")
		console.Warning("- Revert all modified files to their original state")
		console.Warning("- Recreate all deleted files")
		console.Warning("")
		console.Warning("Are you sure you want to continue?")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" {
			console.Info("Aborted")
			return nil
		}
	}

	// Delete all created files
	for _, path := range fc.CreatedFilePaths {
		err = os.Remove(path)
		if err != nil {
			return console.Error("Failed to delete file \"%s\": %s", path, err)
		}
	}

	// Build hash map for overridden files (modified + deleted)
	overrideHashMap := make(map[string]string)
	overrideFilePaths := append(fc.ModifiedFilePaths, fc.DeletedFilePaths...)
	for _, path := range overrideFilePaths {
		hash := commit.HashMap[path]
		overrideHashMap[path] = hash
	}

	// Download remote versions of modified and deleted files
	dataMap, err := storj.DownloadBulk(projectConfig.ProjectID, maps.Values(overrideHashMap))
	if err != nil {
		return console.Error("Failed to download files: %s", err)
	}

	// Write files to disk
	for hash, data := range dataMap {
		// Get file path from hash (reverse lookup)
		path := util.ReverseLookup(overrideHashMap, hash)

		if path == "" {
			return console.Error("Failed to find downloaded file with hash %s", hash)
		}

		// Write file
		err = ioutil.WriteFile(path, data, 0644)
		if err != nil {
			return console.Error("Failed to write file \"%s\" (hash %s): %s", path, hash, err)
		}
	}

	return nil
}
