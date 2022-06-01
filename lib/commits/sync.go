package commits

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/projects"
	"github.com/joshnies/decent/lib/storage"
	"github.com/joshnies/decent/models"
	"golang.org/x/exp/maps"
)

// Sync to a specific commit.
func SyncToCommit(gc models.GlobalConfig, projectConfig models.ProjectConfig, commitIndex int, confirm bool) error {
	console.Verbose("Getting current commit...")
	httpClient := &http.Client{}

	// Get current commit
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/projects/%s/commits/index/%d", config.I.API.Host, projectConfig.ProjectID, projectConfig.CurrentCommitIndex), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	commitRes, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(commitRes); err != nil {
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
		req, err = http.NewRequest("GET", fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, projectConfig.CurrentBranchID), nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
		res, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
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
		req, err = http.NewRequest("GET", fmt.Sprintf("%s/projects/%s/commits/index/%d", config.I.API.Host, projectConfig.ProjectID, commitIndex), nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
		res, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			return err
		}
		defer res.Body.Close()

		// Parse commit
		err = json.NewDecoder(res.Body).Decode(&toCommit)
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
	filesToOverride := []string{}
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
			filesToOverride = append(filesToOverride, key)
		}

		// Add new file to download map
		downloadMap[key] = hash
	}

	confirmed := false

	// Warn user about local file overrides and prompt to continue
	if len(filesToOverride) > 0 && confirm {
		console.Warning("The following files are not mergeable and will be overridden by remote changes:")

		for _, key := range filesToOverride {
			console.Warning("  %s", key)
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
		console.Warning("Sync to commit #%d (\"%s\")? (y/n)", toCommit.Index, toCommit.Message)
		var answer string
		fmt.Scanln(&answer)

		if strings.ToLower(answer) != "y" {
			console.Info("Aborted")
			return nil
		}
	}

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
