package vcscmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TwiN/go-color"
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
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
	reqUrl := fmt.Sprintf("%s/projects/%s/branches?join_commit=true", config.I.VCS.ServerHost, projectConfig.ProjectName)
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
		isCurrentBranch := projectConfig.CurrentBranchName == branch.ID

		currentNote := ""
		if isCurrentBranch {
			currentNote = " (current)"
		}

		fmt.Printf(color.InBold(color.InCyan("%s%s:"))+" commit #%d\n", branch.Name, currentNote, branch.Commit.Index)
	}

	return nil
}
