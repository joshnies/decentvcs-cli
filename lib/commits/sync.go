package commits

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/models"
	"golang.org/x/exp/maps"
)

// Sync to a specific commit.
func SyncToCommit(gc models.GlobalConfig, projectConfig models.ProjectConfig, commitIndex int) error {
	// Get current commit
	commitRes, err := httpw.Get(api.BuildURLf("projects/%s/commits/index/%d", projectConfig.ProjectID, projectConfig.CurrentCommitIndex), gc.Auth.AccessToken)
	if err != nil {
		return err
	}
	defer commitRes.Body.Close()

	// Parse commit
	var currentCommit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&currentCommit)
	if err != nil {
		return console.Error(constants.ErrMsgInternal)
	}

	// Get specified commit ID from args; default to latest commit
	var toCommit models.Commit

	if commitIndex == 0 {
		// Get current branch with latest commit
		res, err := httpw.Get(api.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID), gc.Auth.AccessToken)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		// Parse commit
		var branchwc models.BranchWithCommit
		err = json.NewDecoder(res.Body).Decode(&branchwc)
		if err != nil {
			return console.Error(constants.ErrMsgInternal)
		}

		toCommit = branchwc.Commit
	} else {
		// Validate commit index
		if commitIndex <= 0 {
			return console.Error("Invalid commit index. Must be a positive integer.")
		}

		// Get user-specified commit
		commitRes, err = httpw.Get(api.BuildURLf("projects/%s/commits/index/%d", projectConfig.ProjectID, commitIndex), gc.Auth.AccessToken)
		if err != nil {
			return err
		}
		defer commitRes.Body.Close()

		// Parse commit
		err = json.NewDecoder(commitRes.Body).Decode(&toCommit)
		if err != nil {
			return console.Error(constants.ErrMsgInternal)
		}
	}

	// Return if commit is the same as current commit
	if toCommit.Index == projectConfig.CurrentCommitIndex {
		console.Info("You are already on this commit")
		return nil
	}

	// Get keys for new files by comparing hash maps
	downloadMap := make(map[string]string)
	overriddenFiles := []string{}
	for key, hash := range toCommit.HashMap {
		if curHash, ok := currentCommit.HashMap[key]; !ok {
			// File is new from last commit. Check if it exists in current changes
			if _, err := os.Stat(key); err == nil {
				// File exists in current changes. Compare hashes
				if hash != curHash {
					// File is changed in local and is different from remote, there's a conflict!
					// TODO: Add file to list of conflicts and try to create merge remote file into local file
					overriddenFiles = append(overriddenFiles, key)
				}
			}

			// Add file to list of files to download
			// filesToDownload = append(filesToDownload, key)
			downloadMap[key] = hash
		}
	}

	// Warn user about overridden files and prompt to continue
	// TODO: Implement merge attempt instead for non-binary files
	if len(overriddenFiles) > 0 {
		console.Warning("The following files will be overridden by remote changes:")

		for _, key := range overriddenFiles {
			console.Warning("\t%s", key)
		}

		console.Warning("Are you sure you want to continue? (y/n)")
		var answer string
		fmt.Scanln(&answer)

		if strings.ToLower(answer) != "y" {
			console.Info("Aborted")
			return nil
		}
	}

	// Get keys for deleted files by comparing hash maps
	filesToDelete := []string{}
	for key, hash := range currentCommit.HashMap {
		if _, ok := toCommit.HashMap[key]; !ok {
			// File is deleted from last commit. Add to list of files to delete if it doesn't exist in current changes
			curHash, err := projects.GetFileHash(key)
			if err != nil {
				return err
			}
			if curHash == hash {
				// File is unchanged from current commit remote, add to list of files to delete
				filesToDelete = append(filesToDelete, key)
			}
		}
	}

	// Prompt user to confirm sync
	console.Info("Are you sure you want to sync to commit #%d? (y/n)", toCommit.Index)
	var answer string
	fmt.Scanln(&answer)

	if strings.ToLower(answer) != "y" {
		console.Info("Aborted")
		return nil
	}

	// Download new files
	dataMap, err := storj.DownloadBulk(projectConfig.ProjectID, maps.Values(downloadMap))
	if err != nil {
		return err
	}

	for _, hash := range maps.Keys(dataMap) {
		// Write file to local filesystem
		var path string
		for p, h := range downloadMap {
			if hash == h {
				path = p
				break
			}
		}

		if path == "" {
			return console.Error("Failed to download file with hash %s", hash)
		}

		err = ioutil.WriteFile(path, dataMap[hash], 0644)
		if err != nil {
			return console.Error("Failed to write file (%s) after downloading: %s", path, err)
		}
	}

	// Delete deleted files
	for _, key := range filesToDelete {
		err = os.Remove(key)
		if err != nil {
			return console.Error("Failed to delete file %s; %s", key, err)
		}
	}

	// Update current commit ID in project config
	projectConfig.CurrentCommitIndex = toCommit.Index
	_, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	return nil
}