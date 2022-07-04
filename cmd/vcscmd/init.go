package vcscmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
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

	// Get project blob ("team-blob/project-blob")
	blob := strings.TrimSpace(c.Args().First())
	if blob == "" {
		// Default to directory name
		blob = filepath.Base(absPath)
	}

	// TODO: Validate blob via regex

	// Create project in API
	httpClient := http.Client{}
	bodyJson, _ := json.Marshal(map[string]string{"blob": blob})
	reqUrl := fmt.Sprintf("%s/projects", config.I.VCS.ServerHost)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
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
		log.Fatalf("Project \"%s\" was created without a default branch. This should never happen! Please submit this as a bug!", blob)
	}

	currentBranch := project.Branches[0]

	// Create project file
	projectFileData := models.ProjectConfig{
		ProjectID:          project.ID,
		CurrentBranchID:    currentBranch.ID,
		CurrentCommitIndex: currentBranch.Commit.Index,
	}
	vcs.SaveProjectConfig(absPath, projectFileData)

	console.Info("Created project %s", project.Blob)
	return nil
}
