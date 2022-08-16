package cmd

import (
	"strconv"

	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/console"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/urfave/cli/v2"
)

// Sync local project to a commit
func Sync(c *cli.Context) error {
	auth.HasToken()

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	if projectConfig.CurrentCommitIndex <= 0 {
		return console.Error("Current commit index is invalid. Please check your project config file.")
	}

	commitIndex, _ := strconv.Atoi(c.Args().Get(0))
	return vcs.SyncToCommit(projectConfig, commitIndex, !c.Bool("no-confirm"))
}
