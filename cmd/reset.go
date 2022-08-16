package cmd

import (
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/urfave/cli/v2"
)

// Command for resetting all changes on local machine.
func Reset(c *cli.Context) error {
	auth.HasToken()
	return vcs.ResetChanges(!c.Bool("no-confirm"))
}
