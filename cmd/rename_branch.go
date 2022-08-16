package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/TwiN/go-color"
	"github.com/joshnies/dvcs/config"
	"github.com/joshnies/dvcs/constants"
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/console"
	"github.com/joshnies/dvcs/lib/httpvalidation"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/joshnies/dvcs/models"
	"github.com/urfave/cli/v2"
)

// Rename the specified branch.
func RenameBranch(c *cli.Context) error {
	auth.HasToken()

	// Get args
	oldName := c.Args().First()
	if oldName == "" {
		return console.Error("Please specify the branch to rename")
	}

	newName := c.Args().Get(1)
	if newName == "" {
		return console.Error("Please specify the new name for the branch")
	}

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get specified branch (for validation purposes)
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectSlug, oldName)
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

	// Parse response
	var branch models.Branch
	err = json.NewDecoder(res.Body).Decode(&branch)
	if err != nil {
		return err
	}

	// Rename branch
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectSlug, oldName)
	bodyJson, _ := json.Marshal(map[string]string{"name": newName})
	req, err = http.NewRequest("PUT", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Add(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	req.Header.Add("Content-Type", "application/json")
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// If current branch, update current branch name in project config
	if projectConfig.CurrentBranchName == oldName {
		projectConfig.CurrentBranchName = newName
		projectConfigPath, err := vcs.GetProjectConfigPath()
		if err != nil {
			return err
		}

		if _, err = vcs.SaveProjectConfig(filepath.Dir(projectConfigPath), projectConfig); err != nil {
			return err
		}
	}

	fmt.Printf("Renamed branch %s to %s\n", color.InRed(oldName), color.InGreen(newName))
	return nil
}
