package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
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
		fmt.Printf(color.InBold(color.InCyan("%s:"))+" commit #%d\n", branch.Name, branch.Commit.Index)
	}

	return nil
}
