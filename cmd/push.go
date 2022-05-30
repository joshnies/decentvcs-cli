package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/lib/projects"
	"github.com/joshnies/quanta/lib/storage"
	"github.com/joshnies/quanta/models"
	"github.com/urfave/cli/v2"
)

type PushOptions struct {
	Message string
	Confirm bool
}

func WithMessage(message string) func(*PushOptions) {
	return func(o *PushOptions) {
		o.Message = message
	}
}

func WithNoConfirm() func(*PushOptions) {
	return func(o *PushOptions) {
		o.Confirm = false
	}
}

// Push local changes to remote
func Push(c *cli.Context, opts ...func(*PushOptions)) error {
	gc := auth.Validate()

	// Build options
	o := &PushOptions{
		Message: c.String("message"),
		Confirm: !c.Bool("no-confirm"),
	}

	if o.Message == "" {
		o.Message = "No message"
	}

	for _, opt := range opts {
		opt(o)
	}

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ current commit
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, projectConfig.CurrentBranchID)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Make sure user is synced with remote before continuing
	if currentBranch.Commit.Index != projectConfig.CurrentCommitIndex {
		return console.Error("You are not synced with the remote. Please run `quanta sync`.")
	}

	// Detect local changes
	console.Info("Detecting changes...")
	startTime := time.Now()
	// TODO: Use user-provided project path if available
	fc, err := projects.DetectFileChanges(".", currentBranch.Commit.HashMap)
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

	// Prompt user for confirmation
	if o.Confirm {
		console.Warning("Push these changes to \"%s\" branch? (y/n)", currentBranch.Name)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" {
			console.Info("Aborted")
			return nil
		}
	}

	// TODO: Create commit after uploads are complete?
	console.Verbose("Creating commit...")
	startTime = time.Now()

	// Create commit in database
	bodyJson, _ := json.Marshal(map[string]any{
		"branch_id":      projectConfig.CurrentBranchID,
		"message":        o.Message,
		"created_files":  fc.CreatedFilePaths,
		"modified_files": fc.ModifiedFilePaths,
		"deleted_files":  fc.DeletedFilePaths,
		"hash_map":       fc.HashMap,
	})
	reqUrl = fmt.Sprintf("%s/projects/%s/commits", config.I.API.Host, projectConfig.ProjectID)
	req, err = http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(res.Body).Decode(&commit)
	if err != nil {
		return err
	}

	console.Verbose("Commit #%d (ID: %s) created successfully", commit.Index, commit.ID)
	console.Verbose("Updating current commit ID in project config...")

	// Update current commit ID in project config
	projectConfig.CurrentCommitIndex = commit.Index
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
		err = storage.UploadMany(projectConfig.ProjectID, uploadHashMap)
		if err != nil {
			return err
		}
	}

	timeElapsed = time.Since(startTime).Truncate(time.Microsecond)
	console.Success("Commit #%d pushed in %s", commit.Index, timeElapsed)
	return nil
}
