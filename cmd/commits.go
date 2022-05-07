package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TwiN/go-color"
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/models"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"storj.io/uplink"
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

	// TODO: Make sure user is synced with remote before continuing

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

	// Detect local changes
	console.Info("Detecting changes...")
	startTime := time.Now()
	changes, hashMap, err := projects.DetectFileChanges(currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	timeElapsed := time.Since(startTime).Truncate(time.Microsecond)

	// If there are no changes, exit
	if len(changes) == 0 {
		console.Info("No changes detected (took %s)", timeElapsed)
		return nil
	}

	// Create slice for each change type
	createdFileChanges := lo.Filter(changes, func(change models.FileChange, _ int) bool {
		return change.Type == models.FileWasCreated
	})

	modifiedFileChanges := lo.Filter(changes, func(change models.FileChange, _ int) bool {
		return change.Type == models.FileWasModified
	})

	deletedFileChanges := lo.Filter(changes, func(change models.FileChange, _ int) bool {
		return change.Type == models.FileWasDeleted
	})

	// Get only file paths for each change type
	createdFilePaths := lo.Map(createdFileChanges, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	modifiedFilePaths := lo.Map(modifiedFileChanges, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	deletedFilePaths := lo.Map(deletedFileChanges, func(entry models.FileChange, _ int) string {
		return entry.Path
	})

	console.Info("%d changes found (took %s). Pushing...", len(changes), timeElapsed)
	console.Verbose("Created: %d", len(createdFilePaths))
	console.Verbose("Modified: %d", len(modifiedFilePaths))
	console.Verbose("Deleted: %d", len(deletedFilePaths))

	// Pull modified files from remote
	var downloads []*uplink.Download

	if len(modifiedFilePaths) > 0 {
		console.Verbose("Downloading latest version of %d modified files...", len(modifiedFilePaths))
		downloads, err = storj.DownloadBulk(projectConfig, modifiedFilePaths)
		if err != nil {
			return err
		}

		console.Verbose("%d files downloaded successfully", len(downloads))
	}

	// TODO: Create patch files (if files exist in remote)
	// TODO: Compress patch files (if any were created)

	console.Verbose("Creating commit...")

	// Create commit in database
	apiUrl = api.BuildURLf("projects/%s/commits", projectConfig.ProjectID)
	bodyJson, _ := json.Marshal(map[string]any{
		"branch_id":      projectConfig.CurrentBranchID,
		"message":        msg,
		"snapshot_paths": createdFilePaths,
		"patch_paths":    modifiedFilePaths,
		"deleted_paths":  deletedFilePaths,
		"hash_map":       hashMap,
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

	// TODO: Upload patch files to storage (if any patch files were created)

	console.Verbose("Uploading %d created files as snapshots...", len(createdFilePaths))

	// TODO: Compress snapshots

	// Upload created files to storage as snapshots
	prefix := fmt.Sprintf("%s/%s", projectConfig.ProjectID, commit.ID)
	err = storj.UploadBulk(prefix, createdFilePaths)
	if err != nil {
		return err
	}

	console.Verbose("Successfully uploaded new files")
	console.Success("Successful")
	return nil
}

// Pull latest changes from remote
func Pull(c *cli.Context) error {
	auth.Validate()

	println("TODO")
	return nil
}

// Print list of current changes
func GetChanges(c *cli.Context) error {
	gc := auth.Validate()

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

	// Detect local changes
	changes, _, err := projects.DetectFileChanges(currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	// If there are no changes, exit
	if len(changes) == 0 {
		console.Info("No changes detected")
		return nil
	}

	// Print changes
	console.Info("%d changes found:", len(changes))

	for _, change := range changes {
		switch change.Type {
		case models.FileWasCreated:
			fmt.Printf(color.Ize(color.Green, "  + %s\n"), change.Path)
		case models.FileWasModified:
			// TODO: Print lines added and removed
			fmt.Printf(color.Ize(color.Cyan, "  * %s\n"), change.Path)
		case models.FileWasDeleted:
			fmt.Printf(color.Ize(color.Red, "  - %s\n"), change.Path)
		}
	}

	return nil
}
