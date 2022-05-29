package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/constants"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

// Initialize a new project on local system and in the database.
func Init(c *cli.Context) error {
	gc := auth.Validate()

	// TODO: Add "name" option (defaults to current directory name)

	// Get absolute file path
	path := strings.TrimSpace(c.Args().First())
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

	// Get project name from absolute path
	name := filepath.Base(absPath)

	// Create project in API
	httpClient := http.Client{}
	bodyJson, _ := json.Marshal(map[string]string{"name": name})
	reqUrl := fmt.Sprintf("%s/projects", config.I.API.Host)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	req.Header.Set("Content-Type", "application/json")
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
		log.Fatalf("Project \"%s\" was created without a default branch. This should never happen! Please submit this as a bug!", name)
	}

	currentBranch := project.Branches[0]

	console.Verbose("Project ID: %s", project.ID)
	console.Verbose("Current branch ID: %s", currentBranch.ID)
	console.Verbose("Current commit index: %d", currentBranch.Commit.Index)

	// Create project file
	projectFileData := models.ProjectConfig{
		ProjectID:          project.ID,
		CurrentBranchID:    currentBranch.ID,
		CurrentCommitIndex: currentBranch.Commit.Index,
	}
	config.SaveProjectConfig(absPath, projectFileData)

	console.Info("Created project \"%s\"", name)
	return nil
}
