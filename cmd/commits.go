package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib"
	"github.com/joshnies/qc-cli/models"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"storj.io/uplink"
)

// Push local changes to remote
func Push(c *cli.Context) error {
	// Make sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// TODO: Make sure user is synced with remote before continuing

	// Detect local changes
	changes, history, err := lib.DetectFileChanges()
	if err != nil {
		return err
	}

	// If there are no changes, exit
	if len(changes) == 0 {
		lib.Log(lib.LogOptions{
			Level: lib.Info,
			Str:   "No changes detected",
		})
		return nil
	}

	// Create slice for each change type
	createdFileChanges := lo.Filter(changes, func(change models.FileChange, _ int) bool {
		return change.Type == models.FileWasCreated
	})

	modifiedFileChanges := lo.Filter(changes, func(change models.FileChange, _ int) bool {
		return change.Type == models.FileWasModified
	})

	deletedFileChanges := lo.Filter(changes, func(change models.FileChange, _ int) bool {
		return change.Type == models.FileWasDeleted
	})

	// Get only file paths for each change type
	allFilePaths := lo.Map(changes, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	createdFilePaths := lo.Map(createdFileChanges, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	modifiedFilePaths := lo.Map(modifiedFileChanges, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	deletedFilePaths := lo.Map(deletedFileChanges, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	lib.Log(lib.LogOptions{
		Level:      lib.Info,
		Str:        "%d changes detected",
		Vars:       []interface{}{len(allFilePaths)},
		VerboseStr: "Change counts:\n\tCreated: %d\n\tModified: %d\n\tDeleted: %d",
		VerboseVars: []interface{}{
			len(createdFilePaths),
			len(modifiedFilePaths),
			len(deletedFilePaths),
		},
	})

	// Pull modified files from remote
	var downloads []*uplink.Download

	if len(modifiedFilePaths) > 0 {
		lib.Log(lib.LogOptions{
			Level: lib.Verbose,
			Str:   "Downloading latest version of %d modified files...",
			Vars:  []interface{}{len(modifiedFilePaths)},
		})

		downloads, err = lib.DownloadBulk(projectConfig, modifiedFilePaths)
		if err != nil {
			return err
		}

		lib.Log(lib.LogOptions{
			Level: lib.Verbose,
			Str:   "%d files downloaded",
			Vars:  []interface{}{len(downloads)},
		})
	}

	// TODO: Create patch files (if files exist in remote)
	// TODO: Compress patch files (if any were created)

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Creating commit...",
	})

	// Create commit in database
	bodyJson, _ := json.Marshal(map[string]any{"name": "test", "snapshot_paths": createdFilePaths})
	body := bytes.NewBuffer(bodyJson)
	commitRes, err := http.Post(lib.BuildURLf("projects/%s/commits", projectConfig.ProjectID), "application/json", body)
	if err != nil {
		return err
	}
	defer commitRes.Body.Close()

	if commitRes.StatusCode != http.StatusOK {
		return lib.Log(lib.LogOptions{
			Level:       lib.Error,
			Str:         "Failed to create commit",
			VerboseStr:  "Failed to create commit via API (status: %s)",
			VerboseVars: []interface{}{commitRes.Status},
		})
	}

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&commit)
	if err != nil {
		return err
	}

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Commit %s created successfully",
		Vars:  []interface{}{commit.ID},
	})

	// TODO: Upload patch files to storage (if any patch files were created)

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Uploading %d new files as snapshots...",
		Vars:  []interface{}{len(createdFilePaths)},
	})

	// TODO: Compress snapshots

	// Upload created files to storage as snapshots
	prefix := fmt.Sprintf("%s/%s", projectConfig.ProjectID, commit.ID)
	err = lib.UploadBulk(prefix, createdFilePaths)
	if err != nil {
		return err
	}

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Successfully uploaded new files",
	})

	if len(deletedFilePaths) > 0 {
		// Create commit file data
		//
		// This file's data will be used later to delete files from users' local projects when
		// pulling changes from remote
		commitFileData := map[string]any{
			"deleted": deletedFilePaths,
		}

		// Parse data into JSON
		commitFileJson, err := json.Marshal(commitFileData)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("%s/%s/%s", projectConfig.ProjectID, commit.ID, constants.CommitFileName)

		// Upload to storage
		err = lib.UploadJSON(key, commitFileJson)
		if err != nil {
			return err
		}
	}

	// Write new history, generated back when we detected changes
	historyJson, _ := json.Marshal(history)
	err = ioutil.WriteFile(constants.HistoryFileName, historyJson, os.ModePerm)
	if err != nil {
		return lib.Log(lib.LogOptions{
			Level:       lib.Error,
			Str:         "Failed to write history file",
			VerboseStr:  "%v",
			VerboseVars: []interface{}{err},
		})
	}

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "History updated successfully. Uploading...",
	})

	// Upload history to storage
	historyFileKey := fmt.Sprintf("%s/%s/%s", projectConfig.ProjectID, commit.ID, constants.HistoryFileName)
	err = lib.UploadJSON(historyFileKey, historyJson)
	if err != nil {
		return lib.Log(lib.LogOptions{
			Level:       lib.Error,
			Str:         "Failed to upload history file",
			VerboseStr:  "%v",
			VerboseVars: []interface{}{err},
		})
	}

	lib.Log(lib.LogOptions{
		Level: lib.Info,
		Str:   "Commit %s successful",
		Vars:  []interface{}{commit.ID},
	})

	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	println("TODO")
	return nil
}
