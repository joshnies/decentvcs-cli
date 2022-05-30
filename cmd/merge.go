package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/lib/projects"
	"github.com/joshnies/quanta/lib/storage"
	"github.com/joshnies/quanta/lib/util"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

// Merge the specified branch into the current branch.
// User must be synced with remote first.
func Merge(c *cli.Context) error {
	gc := auth.Validate()

	// Extract args
	branchName := c.Args().Get(0)
	if branchName == "" {
		return console.Error("Please specify name of branch to merge")
	}

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ current commit
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, projectConfig.CurrentBranchID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Make sure user is synced with remote before continuing
	if currentBranch.Commit.Index != projectConfig.CurrentCommitIndex {
		return console.Error("You are not synced with the remote. Please run `quanta pull`.")
	}

	// Get specified branch w/ commit
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, branchName)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var branchToMerge models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branchToMerge)
	if err != nil {
		return err
	}

	// Detect local changes
	// TODO: Use user-provided project path if available
	fc, err := projects.DetectFileChanges(".", currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	// Detect new files in branch to merge
	createdHashMap := make(map[string]string)
	for path, hash := range branchToMerge.Commit.HashMap {
		if _, ok := fc.HashMap[path]; !ok {
			createdHashMap[path] = hash
		}
	}

	// Get difference between local hash map and the hash map of the branch to merge
	modifiedHashMap := make(map[string]string)
	for path, hash := range fc.HashMap {
		if hash != branchToMerge.Commit.HashMap[path] {
			modifiedHashMap[path] = hash
		}
	}

	combinedHashMap := util.MergeMaps(createdHashMap, modifiedHashMap)

	// Return if no changes detected
	if len(combinedHashMap) == 0 {
		fmt.Println("No changes detected, nothing to merge.")
		return nil
	}

	// Create temp dir for storing downloaded files
	tempDirPath, err := os.MkdirTemp("", "quanta-merge-")
	if err != nil {
		return err
	}

	// Download created and modified files from storage
	console.Verbose("Downloading created & modified files to %s", tempDirPath)
	err = storage.DownloadMany(projectConfig.ProjectID, tempDirPath, combinedHashMap)
	if err != nil {
		return err
	}

	// TODO: Print changes to be merged.
	// For binary files, only show file name and size (compressed).
	// For text-based files, show file name and diff.

	// TODO: Prompt user to confirm merge

	// TODO: Move created files to project dir

	// TODO: Merge modified files

	// TODO: Delete temp dir

	// TODO: Push if `push` flag provided (after user confirmation)

	return nil
}
