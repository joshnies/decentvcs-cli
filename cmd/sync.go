package cmd

import (
	"strconv"

	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/commits"
	"github.com/joshnies/qc/lib/console"
	"github.com/urfave/cli/v2"
)

// Sync local project to a commit
func Sync(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	if projectConfig.CurrentCommitIndex <= 0 {
		return console.Error("Current commit ID is invalid. Please check your project config file.")
	}

	commitIndex, _ := strconv.Atoi(c.Args().Get(0))
	return commits.SyncToCommit(gc, projectConfig, commitIndex, !c.Bool("no-confirm"))
}
