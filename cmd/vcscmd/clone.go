package vcscmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/storage"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

// Clone remote project at default branch to local machine.
func CloneProject(c *cli.Context) error {
	auth.HasToken()

	// Get project blob from first arg
	projectBlob := c.Args().First()
	if projectBlob == "" {
		return console.Error("Please specify a project in the format of \"<author_or_team_alias>/<project_name>\"")
	}

	// Get clone path from second arg
	clonePath, err := filepath.Abs(c.String("path"))
	if err != nil {
		return console.Error("Invalid path")
	}

	// Get branch name
	branchName := c.String("branch")

	// Check if already in project directory
	if _, err := os.Stat(filepath.Join(clonePath, constants.ProjectFileName)); !os.IsNotExist(err) {
		return console.Error("A project already exists in the current directory")
	}

	// Get project
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/blob/%s", config.I.VCS.ServerHost, projectBlob)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse project
	var project models.Project
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return err
	}

	var branch models.BranchWithCommit
	if branchName == "" {
		// Get default branch
		reqUrl = fmt.Sprintf("%s/projects/%s/branches/default?join_commit=true", config.I.VCS.ServerHost, project.ID)
		req, err = http.NewRequest("GET", reqUrl, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
		res, err = httpClient.Do(req)
		if err != nil {
			return err
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			return err
		}
		defer res.Body.Close()

		// Parse branch
		err = json.NewDecoder(res.Body).Decode(&branch)
		if err != nil {
			return err
		}
	} else {
		// Get specified branch
		reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.VCS.ServerHost, project.ID, branchName)
		req, err = http.NewRequest("GET", reqUrl, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.I.Auth.SessionToken))
		res, err = httpClient.Do(req)
		if err != nil {
			return err
		}
		if err = httpvalidation.ValidateResponse(res); err != nil {
			return err
		}
		defer res.Body.Close()

		// Parse branch
		err = json.NewDecoder(res.Body).Decode(&branch)
		if err != nil {
			return err
		}
	}

	if len(maps.Values(branch.Commit.HashMap)) == 0 {
		return console.Error("No committed files found for branch \"%s\"", branch.Name)
	}

	// Create clone directory resursively
	err = os.MkdirAll(clonePath, 0755)
	if err != nil {
		return err
	}

	console.Info("Cloning project \"%s\" with branch \"%s\" into \"%s\"...", project.Name, branch.Name, clonePath)
	console.Verbose("Branch commit index: %d", branch.Commit.Index)

	// Create project config file
	console.Verbose("Creating project config file...")
	projectConfig := models.ProjectConfig{
		ProjectID:          project.ID,
		CurrentBranchID:    branch.ID,
		CurrentCommitIndex: branch.Commit.Index,
	}

	projectConfig, err = vcs.SaveProjectConfig(clonePath, projectConfig)
	if err != nil {
		return err
	}

	console.Verbose("Project config file created")

	// Download all files
	err = storage.DownloadMany(projectConfig.ProjectID, clonePath, branch.Commit.HashMap)
	if err != nil {
		return err
	}

	return nil
}
