package vcscmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TwiN/go-color"
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
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
	reqUrl := fmt.Sprintf("%s/projects/%s", config.I.VCS.ServerHost, projectConfig.ProjectID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
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
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectID, projectConfig.CurrentBranchID)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
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
	reqUrl = fmt.Sprintf("%s/projects/%s/commits/index/%d", config.I.VCS.ServerHost, projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
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
