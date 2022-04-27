package cmd

import (
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib"
	"github.com/urfave/cli/v2"
)

// Push local changes to remote
func Push(c *cli.Context) error {
	// Make sure current directory is a project
	_, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Detect local changes
	changedFiles, err := lib.DetectFileChanges()
	if err != nil {
		return err
	}

	lib.Log(lib.LogOptions{
		Level: lib.Verbose,
		Str:   "%d changes detected",
		Vars:  []interface{}{len(changedFiles)},
	})

	// Pull changed files from remote

	// TODO: Create patch files (if files exist in remote)
	// TODO: Upload patch files to storage (if any patch files were created)
	// TODO: Upload new files to storage (initial snapshots)
	// TODO: Create commit in database
	// TODO: Update history file

	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	println("TODO")
	return nil
}
