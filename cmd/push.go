package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/storage"
	"github.com/decentvcs/cli/lib/system"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
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
	auth.HasToken()

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
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current branch w/ latest commit
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf(
		"%s/projects/%s/branches/%s?join_commit=true",
		config.I.VCS.ServerHost,
		projectConfig.ProjectSlug,
		projectConfig.CurrentBranchName,
	)
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

	// Parse response
	var currentBranch models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Get current commit
	reqUrl = fmt.Sprintf(
		"%s/projects/%s/commits/%d",
		config.I.VCS.ServerHost,
		projectConfig.ProjectSlug,
		projectConfig.CurrentCommitIndex,
	)
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

	// Parse response
	var currentCommit models.Commit
	err = json.NewDecoder(res.Body).Decode(&currentCommit)
	if err != nil {
		return err
	}

	// Get "force" flag
	force := c.Bool("force")

	// Make sure user is synced with remote before continuing
	if currentBranch.Commit.Index != projectConfig.CurrentCommitIndex {
		if force {
			console.Warning("You're about to force push!")
			console.Warning(
				"This will permanently delete all commits and new files ahead of your current commit (#%d) on this "+
					"branch (%s).",
				projectConfig.CurrentCommitIndex,
				currentBranch.Name,
			)
			console.Warning("Continue? (y/n)")

			var answer string
			fmt.Scanln(&answer)
			if answer != "y" {
				console.Info("Aborted")
				return nil
			}
		} else {
			console.ErrorPrint(
				"Your are on commit #%d, but the remote branch points to commit #%d.",
				projectConfig.CurrentCommitIndex,
				currentCommit.Index,
			)
			return console.Error("You can forcefully push your changes by using the --force flag.")
		}
	}

	// Detect local changes
	startTime := time.Now()
	fc, err := vcs.DetectFileChanges(currentCommit.Files)
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

	// Get project for later
	reqUrl = fmt.Sprintf(
		"%s/projects/%s",
		config.I.VCS.ServerHost,
		projectConfig.ProjectSlug,
	)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err = httpClient.Do(req)
	if err != nil {
		return err
	}

	var project models.Project
	err = json.NewDecoder(res.Body).Decode(&project)
	if err != nil {
		return err
	}

	if force {
		// User is force pushing.
		// Delete commits ahead of current commit.
		if err = vcs.DeleteCommitsAheadOfIndex(projectConfig, currentBranch.ID, currentCommit.Index); err != nil {
			return err
		}
	}

	startTime = time.Now()

	var patchFilePaths []string
	if project.EnablePatchRevisions && len(fc.ModifiedFilePaths) > 0 {
		// Download modified files from storage
		tempDirPath := system.GetTempDir()
		modifiedFileHashMap := make(map[string]string)
		for _, filePath := range fc.ModifiedFilePaths {
			modifiedFileHashMap[filePath] = currentCommit.Files[filePath].Hash
		}

		fmt.Printf("Temp dir path: %s\n", tempDirPath)                  // DEBUG
		fmt.Printf("Modified file hash map: %v\n", modifiedFileHashMap) // DEBUG

		err = storage.DownloadMany(projectConfig.ProjectSlug, tempDirPath, modifiedFileHashMap)
		if err != nil {
			return err
		}

		// Generate patches for modified files
		for _, modFilePath := range fc.ModifiedFilePaths {
			oldFilePath := filepath.Join(tempDirPath, modFilePath) // same as mod file, just in temp dir from download above
			patchPath := filepath.Join(tempDirPath, modFilePath+".patch")
			patchFilePaths = append(patchFilePaths, patchPath)

			err := vcs.GenPatchFile(oldFilePath, modFilePath, patchPath)
			if err != nil {
				return err
			}
		}
	}

	// Gather file paths for upload
	modifiedFilePaths := fc.ModifiedFilePaths
	if project.EnablePatchRevisions {
		modifiedFilePaths = patchFilePaths
	}

	filesToUpload := []string{}
	filesToUpload = append(filesToUpload, fc.CreatedFilePaths...)
	filesToUpload = append(filesToUpload, modifiedFilePaths...)

	uploadHashMap := make(map[string]string)

	for _, path := range filesToUpload {
		uploadHashMap[path] = fc.FileDataMap[path].Hash
	}

	// Upload files to storage
	if len(filesToUpload) > 0 {
		err = storage.UploadMany(projectConfig.ProjectSlug, uploadHashMap)
		if err != nil {
			return err
		}
	}

	// Create commit (the team will be charged for any storage space used)
	console.Verbose("Creating commit...")
	bodyJson, _ := json.Marshal(map[string]interface{}{
		"message":        o.Message,
		"created_files":  fc.CreatedFilePaths,
		"modified_files": modifiedFilePaths,
		"deleted_files":  fc.DeletedFilePaths,
		"files":          fc.FileDataMap,
	})
	reqUrl = fmt.Sprintf(
		"%s/projects/%s/branches/%s/commit",
		config.I.VCS.ServerHost,
		projectConfig.ProjectSlug,
		projectConfig.CurrentBranchName,
	)
	req, err = http.NewRequest("POST", reqUrl, bytes.NewBuffer(bodyJson))
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
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

	console.Verbose("Commit #%d created successfully", commit.Index)
	console.Verbose("Updating current commit index in project config...")

	// Update current commit index in project config
	projectConfig.CurrentCommitIndex = commit.Index
	projectConfigPath, err := vcs.GetProjectConfigPath()
	if err != nil {
		return err
	}

	if _, err = vcs.SaveProjectConfig(filepath.Dir(projectConfigPath), projectConfig); err != nil {
		return err
	}

	timeElapsed = time.Since(startTime).Truncate(time.Microsecond)
	console.Success("Commit #%d pushed in %s", commit.Index, timeElapsed)
	return nil
}
