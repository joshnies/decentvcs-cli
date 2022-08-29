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

// Print info for the current project, branch, and commit
func PrintStatus(c *cli.Context) error {
	auth.HasToken()

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get project
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s", config.I.VCS.ServerHost, projectConfig.ProjectSlug)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Parse project
	var project models.Project
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return console.Error("Failed to parse project: %s", err)
	}
	res.Body.Close()

	// Get branch
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectSlug, projectConfig.CurrentBranchName)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Parse branch
	var branch models.Branch
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return console.Error("Failed to parse branch: %s", err)
	}
	res.Body.Close()

	// Get commit
	reqUrl = fmt.Sprintf("%s/projects/%s/commits/%d", config.I.VCS.ServerHost, projectConfig.ProjectSlug, projectConfig.CurrentCommitIndex)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(res.Body).Decode(&commit)
	if err != nil {
		return console.Error("Failed to parse commit: %s", err)
	}
	res.Body.Close()

	fmt.Printf(color.Ize(color.Cyan, "Project: ")+"%s (%s)\n", project.Name, project.ID)
	fmt.Printf(color.Ize(color.Cyan, "Branch:  ")+"%s (%s)\n", branch.Name, branch.ID)
	fmt.Printf(color.Ize(color.Cyan, "Commit:  ")+"#%d (%s)\n", commit.Index, commit.ID)

	return nil
}