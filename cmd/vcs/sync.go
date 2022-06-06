package vcs

import (
	"strconv"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/commits"
	"github.com/joshnies/decent/lib/console"
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
