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
	changedFiles, modTimes, err := lib.DetectFileChanges()
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

	// TODO: Calculate which files need patches and which are new (snapshots)
	// TODO: Create patch files (if files exist in remote)

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Creating commit...",
	})

	// Create commit in database
	bodyJson, _ := json.Marshal(map[string]any{"name": "test", "snapshot_paths": changedFiles})
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
		Str:   "Commit %s created successfully. Downloading files...",
		Vars:  []interface{}{commit.ID},
	})

	// TODO: Upload patch files to storage (if any patch files were created)

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "Uploading %d new files as snapshots...",
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
		Str:   "Snapshot uploads successful. Updating history...",
	})

	// Read history file
	history, err := lib.ReadHistory()
	if err != nil {
		return err
	}

	// Update history
	for i, fpath := range changedFiles {
		history[fpath] = modTimes[i]
	}

	// Write new history
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
		Str:   "History updated successfully",
	})

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
