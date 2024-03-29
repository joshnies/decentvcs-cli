package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/storage"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

// Clone remote project at default branch to local machine.
func CloneProject(c *cli.Context) error {
	auth.HasToken()

	// Get project blob from first arg
	slug := c.Args().First()
	if slug == "" {
		return console.Error("Please specify a project in the format \"<team_name>/<project_name>\"")
	}

	// Validate slug
	if matched, _ := regexp.MatchString(constants.ProjectSlugRegex, slug); !matched {
		return console.Error("Invalid project slug. Please use the format \"<team_name>/<project_name>\"")
	}

	projectName := strings.Split(slug, "/")[1]

	// Get clone path from second arg
	cloneDirRel := c.Args().Get(1)
	if cloneDirRel == "" {
		cloneDirRel = "." // current directory
	}

	cloneDirAbs, err := filepath.Abs(cloneDirRel)
	if err != nil {
		return console.Error("Invalid clone path")
	}

	clonePath := filepath.Join(cloneDirAbs, projectName)

	// Get branch name
	branchName := c.String("branch")

	// Check if clone path is already a project directory
	if _, err := os.Stat(filepath.Join(clonePath, constants.ProjectFileName)); !os.IsNotExist(err) {
		return console.Error("A project already exists at %s", clonePath)
	}

	// Get project
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s", config.I.VCS.ServerHost, slug)
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

	// Parse project
	var project models.Project
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return err
	}

	var branch models.BranchWithCommit
	if branchName == "" {
		// Get default branch
		reqUrl = fmt.Sprintf("%s/projects/%s/branches/default?join_commit=true", config.I.VCS.ServerHost, slug)
		req, err = http.NewRequest("GET", reqUrl, nil)
		if err != nil {
			return err
		}
		req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
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
		reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.VCS.ServerHost, slug, branchName)
		req, err = http.NewRequest("GET", reqUrl, nil)
		if err != nil {
			return err
		}
		req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
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

	if len(maps.Values(branch.Commit.Files)) == 0 {
		return console.Error("No committed files found for branch \"%s\"", branch.Name)
	}

	// Create clone directory resursively
	err = os.MkdirAll(clonePath, 0755)
	if err != nil {
		return err
	}

	console.Info("Cloning project %s with branch \"%s\" into \"%s\"...", slug, branch.Name, clonePath)

	// Create project config file
	console.Verbose("Creating project config file...")
	projectConfig := models.ProjectConfig{
		ProjectSlug:        slug,
		CurrentBranchName:  branch.Name,
		CurrentCommitIndex: branch.Commit.Index,
	}

	projectConfig, err = vcs.SaveProjectConfig(clonePath, projectConfig)
	if err != nil {
		return err
	}

	console.Verbose("Project config file created")

	// Download all files
	hashMap := vcs.FileMapToHashMap(branch.Commit.Files)
	err = storage.DownloadMany(projectConfig, clonePath, hashMap)
	if err != nil {
		return err
	}

	return nil
}
