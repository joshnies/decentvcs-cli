package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
)

// Initialize a new project on local system and in the database.
func Init(c *cli.Context) error {
	auth.HasToken()

	// Get absolute file path
	path := c.String("path")
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Make sure path is not already a project
	if _, err := os.Stat(absPath + "/" + constants.ProjectFileName); err == nil {
		return console.Error("Project already initialized at %s", absPath)
	}

	console.Info("Initializing project in %s...", absPath)

	if path != "." {
		// Create directories if they don't exist
		err = os.MkdirAll(absPath, os.ModePerm)
		if err != nil {
			return console.Error("Failed to create directory %s: %s", absPath, err)
		}
	}

	// Get project slug ("<team_name>/<project_name>")
	slug := strings.TrimSpace(c.Args().First())

	// Validate project slug
	regex := regexp.MustCompile(`^(?:[\w\-\.]+\/)?[\w\-\.]+$`)
	if !regex.MatchString(slug) {
		return console.Error("Invalid project slug; must be in the format \"<team_name>/<project_name>\"")
	}

	// Create project in API
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s", config.I.VCS.ServerHost, slug)
	reqData := models.CreateProjectRequest{
		EnablePatchRevisions: c.Bool("patch"),
	}
	reqJSON, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(reqJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
	var project models.ProjectWithBranchesAndCommit
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return err
	}

	if len(project.Branches) == 0 {
		log.Fatalf("Project \"%s\" was created without a default branch. This should never happen! Please submit this as a bug!", slug)
	}

	currentBranch := project.Branches[0]

	// Create project file
	projectFileData := models.ProjectConfig{
		ProjectSlug:        slug,
		CurrentBranchName:  currentBranch.Name,
		CurrentCommitIndex: currentBranch.Commit.Index,
	}
	vcs.SaveProjectConfig(absPath, projectFileData)

	console.Info("Created project %s", slug)
	return nil
}
