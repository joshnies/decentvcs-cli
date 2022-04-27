package cmd

import (
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib"
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

	// Pull changed files from remote
	_, err = lib.DownloadBulk(projectConfig, changedFiles)
	if err != nil {
		return err
	}

	// TODO: Create patch files (if files exist in remote)
	// TODO: Upload patch files to storage (if any patch files were created)

	// Upload new files to storage (initial snapshots)
	err = lib.UploadBulk(projectConfig, changedFiles)
	if err != nil {
		return err
	}

	// TODO: Create commit in database
	// TODO: Update history file

	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	println("TODO")
	return nil
}
