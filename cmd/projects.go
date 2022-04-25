package cmd

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joshnies/qc-cli/lib"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

func Init(c *cli.Context) error {
	// Get absolute file path
	path := c.Args().First()
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Create directories if they don't exist
	err = os.MkdirAll(absPath, os.ModePerm)
	if err != nil {
		log.Fatalf("\"%s\" is an existing file, aborting...", absPath)
	}

	// Get project name from absolute path
	// TODO: Add cmd option to override project name
	name := filepath.Base(absPath)

	// Create project in API
	bodyJson, _ := json.Marshal(map[string]string{"name": name})
	body := bytes.NewBuffer(bodyJson)
	res, err := http.Post(lib.BuildURL("projects"), "application/json", body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var project models.Project
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return err
	}

	if len(project.Branches) == 0 {
		log.Fatalf("Project \"%s\" was created without a default branch. This should never happen! Please contact us.", name)
	}

	// Create QC project file
	projectFileData := models.ProjectFileData{
		ProjectID:       project.ID,
		CurrentBranchID: project.Branches[len(project.Branches)-1].ID,
	}
	lib.CreateProjectFile(absPath, projectFileData)

	println("Project created successfully!")

	return nil
}
