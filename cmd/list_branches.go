package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TwiN/go-color"
	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
)

// List all branches in project.
func ListBranches(c *cli.Context) error {
	auth.HasToken()

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get all branches in project
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches?join_commit=true", config.I.VCS.ServerHost, projectConfig.ProjectSlug)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse branches
	var branches []models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branches)
	if err != nil {
		return console.Error(constants.ErrInternal)
	}

	// Print branches
	for _, branch := range branches {
		isCurrentBranch := projectConfig.CurrentBranchName == branch.Name

		branchNameFmt := branch.Name + ":"
		if isCurrentBranch {
			branchNameFmt = color.InBold(color.InCyan(fmt.Sprintf("%s (current)", branchNameFmt)))
		} else {
			branchNameFmt = color.InCyan(branchNameFmt)
		}

		fmt.Printf(branchNameFmt+" commit #%d\n", branch.Commit.Index)
	}

	return nil
}