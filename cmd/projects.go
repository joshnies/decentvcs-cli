package cmd

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/models"
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
	if _, err := os.Stat(absPath + "/" + ".qc"); err == nil {
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
	bodyJson, _ := json.Marshal(map[string]string{"name": name})
	body := bytes.NewBuffer(bodyJson)
	res, err := httpw.Post(api.BuildURL("projects"), body, gc.Auth.AccessToken)
	if err != nil {
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
	console.Verbose("Current commit ID: %s", currentBranch.Commit.ID)

	// Create QC project file
	projectFileData := models.ProjectConfig{
		ProjectID:       project.ID,
		CurrentBranchID: currentBranch.ID,
		CurrentCommitID: currentBranch.Commit.ID,
	}
	projects.WriteProjectConfig(absPath, projectFileData)

	console.Info("Project created successfully")
	return nil
}
