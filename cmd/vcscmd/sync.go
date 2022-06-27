package vcscmd

import (
	"strconv"

	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/commits"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/vcs"
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
	return commits.SyncToCommit(projectConfig, commitIndex, !c.Bool("no-confirm"))
}
