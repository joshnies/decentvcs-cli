package cmd

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Push local changes to remote
func Push(c *cli.Context) error {
	gc := auth.Validate()

	// Extract args
	msg := c.Args().Get(0)
	if msg == "" {
		msg = "No message"
	}

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ current commit
	apiUrl := api.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID)
	currentBranchRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}
	defer currentBranchRes.Body.Close()

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(currentBranchRes.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Make sure user is synced with remote before continuing
	if currentBranch.Commit.ID != projectConfig.CurrentCommitID {
		return console.Error("You are not synced with the remote. Please run `qc pull`.")
	}

	// Detect local changes
	console.Info("Detecting changes...")
	startTime := time.Now()
	fc, err := projects.DetectFileChanges(currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	timeElapsed := time.Since(startTime).Truncate(time.Microsecond)

	// If there are no changes, exit
	changeCount := len(fc.CreatedFilePaths) + len(fc.ModifiedFilePaths) + len(fc.DeletedFilePaths)
	if changeCount == 0 {
		console.Info("No changes detected (took %s)", timeElapsed)
		return nil
	}

	console.Info("%d changes found (took %s). Pushing...", changeCount, timeElapsed)
	console.Verbose("Created: %d", len(fc.CreatedFilePaths))
	console.Verbose("Modified: %d", len(fc.ModifiedFilePaths))
	console.Verbose("Deleted: %d", len(fc.DeletedFilePaths))
	console.Verbose("Creating commit...")

	// Create commit in database
	apiUrl = api.BuildURLf("projects/%s/commits", projectConfig.ProjectID)
	bodyJson, _ := json.Marshal(map[string]any{
		"branch_id":      projectConfig.CurrentBranchID,
		"message":        msg,
		"created_files":  fc.CreatedFilePaths,
		"modified_files": fc.ModifiedFilePaths,
		"deleted_files":  fc.DeletedFilePaths,
		"hash_map":       fc.HashMap,
	})
	body := bytes.NewBuffer(bodyJson)
	commitRes, err := httpw.Post(apiUrl, body, gc.Auth.AccessToken)
	if err != nil {
		return err
	}
	defer commitRes.Body.Close()

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&commit)
	if err != nil {
		return err
	}

	console.Verbose("Commit %s created successfully", commit.ID)
	console.Verbose("Updating current commit ID in project config...")

	// Update current commit ID in project config
	projectConfig.CurrentCommitID = commit.ID
	_, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	// Upload snapshots of created and modified files to storage
	uploadHashMap := make(map[string]string)
	filesToUpload := []string{}
	filesToUpload = append(filesToUpload, fc.CreatedFilePaths...)
	filesToUpload = append(filesToUpload, fc.ModifiedFilePaths...)
	for _, path := range filesToUpload {
		uploadHashMap[path] = fc.HashMap[path]
	}

	if len(filesToUpload) > 0 {
		// TODO: Compress files before uploading
		console.Verbose("Uploading %d files...", len(filesToUpload))

		err = storj.UploadBulk(projectConfig.ProjectID, uploadHashMap)
		if err != nil {
			return err
		}

		console.Verbose("Successfully uploaded files")
	}

	console.Success("New commit pushed: %s", commit.ID)
	return nil
}
