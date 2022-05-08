package cmd

import (
	"encoding/json"
	"os"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

// Sync local project to a commit
func Sync(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	if projectConfig.CurrentCommitID == "" {
		// TODO: Add option for user to sync local project with remote, no matter what commit they're on
		return console.Error("Current commit ID is invalid. Please check your project config file.")
	}

	// Get current commit
	commitRes, err := httpw.Get(api.BuildURLf("projects/%s/commits/%s", projectConfig.ProjectID, projectConfig.CurrentCommitID), gc.Auth.AccessToken)
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
	commitID := c.Args().Get(0)

	if commitID == "" {
		// Get latest commit on current branch
		commitRes, err := httpw.Get(api.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID), gc.Auth.AccessToken)
		if err != nil {
			return err
		}
		defer commitRes.Body.Close()

		// Parse commit
		err = json.NewDecoder(commitRes.Body).Decode(&toCommit)
		if err != nil {
			return console.Error(constants.ErrMsgInternal)
		}
	} else {
		// Get user-specified commit
		commitRes, err = httpw.Get(api.BuildURLf("projects/%s/commits/%s", projectConfig.ProjectID, commitID), gc.Auth.AccessToken)
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
	if toCommit.ID == projectConfig.CurrentCommitID {
		console.Info("You are already on this commit")
	}

	// Get files to download
	localPaths := maps.Keys(currentCommit.State)
	downloadMap := map[string]string{} // map of storage key to commit ID

	for key, state := range toCommit.State {
		// Check if file existsLocally locally
		existsLocally := false
		for _, l := range localPaths {
			if key == l {
				existsLocally = true
				break
			}
		}

		// Add path to download list
		if !existsLocally {
			downloadMap[key] = state.HostCommitId
		}
	}

	for _, l := range localPaths {
		// Check if file existsRemotely remotely
		existsRemotely := false
		for _, r := range maps.Keys(toCommit.State) {
			if l == r {
				existsRemotely = true
				break
			}
		}

		// Delete file locally
		if !existsRemotely {
			err = os.Remove(l)
			if err != nil {
				return err
			}
		}
	}

	// Download new files
	for key, hostCommitID := range downloadMap {
		// Get file from storage
		fileData, err := storj.Download(projectConfig.ProjectID, hostCommitID, key)
		if err != nil {
			return err
		}

		// Write file to local storage
		file, err := os.Open(key)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.Write(fileData)
		if err != nil {
			return err
		}
	}

	return nil
}
