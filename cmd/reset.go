package cmd

import (
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/urfave/cli/v2"
)

// Command for resetting all changes on local machine.
func Reset(c *cli.Context) error {
	return projects.ResetChanges(c)
}
