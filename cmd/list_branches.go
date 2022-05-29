package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/constants"
	"github.com/joshnies/quanta/lib/api"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpw"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

// List all branches in project.
func ListBranches(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get all branches in project
	res, err := httpw.Get(api.BuildURLf("projects/%s/branches?join_commit=true", projectConfig.ProjectID), gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse branches
	var branches []models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branches)
	if err != nil {
		return console.Error(constants.ErrMsgInternal)
	}

	// Print branches
	for _, branch := range branches {
		isCurrentBranch := projectConfig.CurrentBranchID == branch.ID

		currentNote := ""
		if isCurrentBranch {
			currentNote = " (current)"
		}

		fmt.Printf(color.InBold(color.InCyan("%s%s:"))+" commit #%d\n", branch.Name, currentNote, branch.Commit.Index)
	}

	return nil
}
