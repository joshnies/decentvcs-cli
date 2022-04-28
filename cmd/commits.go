package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/qc-cli/config"
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

	// Get current branch w/ current commit
	currentBranchRes, err := http.Get(lib.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID))
	if err != nil {
		return err
	}

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(currentBranchRes.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Detect local changes
	changes, hashMap, err := lib.DetectFileChanges(currentBranch.Commit.HashMap)
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
	msg := c.Args().Get(0)
	bodyJson, _ := json.Marshal(map[string]any{
		"branch_id":      projectConfig.CurrentBranchID,
		"message":        msg,
		"snapshot_paths": createdFilePaths,
		"patch_paths":    modifiedFilePaths,
		"deleted_paths":  deletedFilePaths,
		"hash_map":       hashMap,
	})
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

	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	println("TODO")
	return nil
}
