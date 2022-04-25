package cmd

import (
	"github.com/joshnies/qc-cli/config"
	"github.com/urfave/cli/v2"
)

// Push local changes to remote
func Push(c *cli.Context) error {
	// Get project config
	project, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// TODO: Detect local changes
	// TODO: Pull changed files from remote
	// TODO: Create patch files (if files exist in remote)
	// TODO: Upload patch files to storage (if any patch files were created)
	// TODO: Upload new files to storage (initial snapshots)
	// TODO: Create commit in database

	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	println("TODO")
	return nil
}
