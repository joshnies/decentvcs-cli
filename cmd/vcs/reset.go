package vcs

import (
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/corefs"
	"github.com/urfave/cli/v2"
)

// Command for resetting all changes on local machine.
func Reset(c *cli.Context) error {
	gc := auth.Validate()
	return corefs.ResetChanges(gc, !c.Bool("no-confirm"))
}
