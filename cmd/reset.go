package cmd

import (
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/projects"
	"github.com/urfave/cli/v2"
)

// Command for resetting all changes on local machine.
func Reset(c *cli.Context) error {
	gc := auth.Validate()
	return projects.ResetChanges(gc, !c.Bool("no-confirm"))
}
