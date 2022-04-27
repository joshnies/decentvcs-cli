package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Push local changes to remote
func Push(c *cli.Context) error {
	// Make sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Detect local changes
	changedFiles, err := lib.DetectFileChanges()
	if err != nil {
		return err
	}

	lib.Log(lib.LogOptions{
		Level:       lib.Info,
		Str:         "%d changes detected",
		Vars:        []interface{}{len(changedFiles)},
		VerboseStr:  "Files changed: %s",
		VerboseVars: []interface{}{changedFiles},
	})

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Creating commit...",
	})

	// Create commit in database
	commitRes, err := http.Post(lib.BuildURLf("projects/%s/commits", projectConfig.ProjectID), "application/json", nil)
	if err != nil {
		return err
	}
	defer commitRes.Body.Close()

	if commitRes.StatusCode != http.StatusCreated {
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

	// Pull changed files from remote
	downloads, err := lib.DownloadBulk(projectConfig, changedFiles)
	if err != nil {
		return err
	}

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "%d files downloaded",
		Vars:  []interface{}{len(downloads)},
	})

	// TODO: Create patch files (if files exist in remote)
	// TODO: Upload patch files to storage (if any patch files were created)

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Uploading %d new files...",
		Vars:  []interface{}{len(changedFiles)},
	})

	// Upload new files to storage (initial snapshots)
	prefix := fmt.Sprintf("%s/%s", projectConfig.ProjectID, commit.ID)
	err = lib.UploadBulk(prefix, changedFiles)
	if err != nil {
		return err
	}

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Uploads successful",
	})

	// Update commit with snapshot & patch file paths
	bodyJson, _ := json.Marshal(map[string]any{"snapshot_paths": changedFiles})
	body := bytes.NewBuffer(bodyJson)
	updateRes, err := http.Post(lib.BuildURLf("projects/%s/commits/%s", projectConfig.ProjectID, commit.ID), "application/json", body)
	if err != nil {
		return err
	}

	if updateRes.StatusCode != http.StatusOK {
		return lib.Log(lib.LogOptions{
			Level:       lib.Error,
			Str:         "Failed to commit changes",
			VerboseStr:  "Failed to update commit via API (status: %s)",
			VerboseVars: []interface{}{updateRes.Status},
		})
	}

	// TODO: Update history file

	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	println("TODO")
	return nil
}
