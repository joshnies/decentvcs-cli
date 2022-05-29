package commits

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/constants"
	"github.com/joshnies/quanta/lib/api"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpw"
	"github.com/joshnies/quanta/lib/projects"
	"github.com/joshnies/quanta/lib/storage"
	"github.com/joshnies/quanta/models"
	"golang.org/x/exp/maps"
)

// Sync to a specific commit.
func SyncToCommit(gc models.GlobalConfig, projectConfig models.ProjectConfig, commitIndex int, confirm bool) error {
	console.Verbose("Getting current commit...")

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
		console.Verbose("Getting current branch with latest commit...")

		// Get current branch with latest commit
		res, err := httpw.Get(api.BuildURLf("projects/%s/branches/%s?join_commit=true", projectConfig.ProjectID, projectConfig.CurrentBranchID), gc.Auth.AccessToken)
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
		console.Verbose("Getting specified commit with index %d...", commitIndex)

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
	console.Verbose("\n\"to\" commit hash map:\n%v", toCommit.HashMap)
	console.Verbose("\nCurrent commit hash map:\n%v\n", currentCommit.HashMap)

	downloadMap := make(map[string]string)
	overriddenFiles := []string{}
	for key, hash := range toCommit.HashMap {
		if curHash, ok := currentCommit.HashMap[key]; ok {
			// File exists in both commits
			//
			// If file is modified, add to download map
			if hash != curHash {
				downloadMap[key] = hash
			}
			continue
		}

		// File is new from last commit
		//
		// Add to override list if it exists in local changes
		if _, err := os.Stat(key); err == nil {
			overriddenFiles = append(overriddenFiles, key)
		}

		// Add new file to download map
		downloadMap[key] = hash
	}

	confirmed := false

	// Warn user about overridden files and prompt to continue
	// TODO: Implement merge attempt instead for non-binary files
	if len(overriddenFiles) > 0 && confirm {
		console.Warning("The following files will be overridden by remote changes:")

		for _, key := range overriddenFiles {
			console.Warning("\t%s", key)
		}

		console.Warning("Continue? (y/n)")
		var answer string
		fmt.Scanln(&answer)

		if strings.ToLower(answer) != "y" {
			console.Info("Aborted")
			return nil
		}

		confirmed = true
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

	console.Verbose("\nFiles to delete: %v", filesToDelete)

	if len(maps.Keys(downloadMap)) == 0 && len(filesToDelete) == 0 {
		console.Info("Your local changes are equivalent to the commit you are syncing to.")
		console.Info("Aborted")
		return nil
	}

	// Prompt user to confirm sync
	if confirm && !confirmed {
		console.Warning("Sync to commit #%d? (y/n)", toCommit.Index)
		var answer string
		fmt.Scanln(&answer)

		if strings.ToLower(answer) != "y" {
			console.Info("Aborted")
			return nil
		}
	}

	// Download new files
	if len(maps.Keys(downloadMap)) > 0 {
		err := storage.DownloadMany(projectConfig.ProjectID, ".", downloadMap)
		if err != nil {
			return err
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

	console.Info("Synced to commit #%d", toCommit.Index)

	return nil
}
