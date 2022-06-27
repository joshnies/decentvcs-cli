package vcscmd

import (
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/corefs"
	"github.com/urfave/cli/v2"
)

// Command for resetting all changes on local machine.
func Reset(c *cli.Context) error {
	auth.HasToken()
	return corefs.ResetChanges(!c.Bool("no-confirm"))
}
