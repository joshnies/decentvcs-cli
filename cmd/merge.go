package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/lib/storage"
	"github.com/decentvcs/cli/lib/system"
	"github.com/decentvcs/cli/lib/util"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
	"github.com/xyproto/binary"
)

// Merge the specified branch into the current branch.
//
// NOTE: User does not need to be synced with remote first, since they may be force pushing a local
// merge to remote.
func Merge(c *cli.Context) error {
	auth.HasToken()

	// Extract args
	branchName := c.Args().Get(0)
	if branchName == "" {
		return console.Error("Please specify name of branch to merge")
	}

	confirm := !c.Bool("yes")
	push := c.Bool("push")

	// Get project config, implicitly making sure current directory is a project
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Calculate local hash map
	localHashMap, err := vcs.CalculateHashes(".")
	if err != nil {
		return err
	}

	// Get current branch
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/branches/%s", config.I.VCS.ServerHost, projectConfig.ProjectSlug, projectConfig.CurrentBranchName)
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
	var currentBranch models.Branch
	err = json.NewDecoder(res.Body).Decode(&currentBranch)
	if err != nil {
		return err
	}

	// Get specified branch w/ commit
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.VCS.ServerHost, projectConfig.ProjectSlug, branchName)
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
	var branchToMerge models.BranchWithCommit
	err = json.NewDecoder(res.Body).Decode(&branchToMerge)
	if err != nil {
		return err
	}

	// Detect movable files, which will simply be moved to the local project, overriding the current
	// versions.
	mvHashMap := make(map[string]string)
	for path, f := range branchToMerge.Commit.Files {
		if _, ok := localHashMap[path]; !ok {
			mvHashMap[path] = f.Hash
		}
	}

	// Detect mergable files
	mergeHashMap := make(map[string]string)
	for path, hash := range localHashMap {
		newHash := branchToMerge.Commit.Files[path].Hash
		if hash != newHash {
			// Check if file is binary data
			isBinary, err := binary.File(path)
			if err != nil {
				return err
			}

			if isBinary {
				// Binary, file cannot be merged
				mvHashMap[path] = newHash
			} else {
				// Non-binary, file can be merged
				mergeHashMap[path] = newHash
			}
		}
	}

	combinedHashMap := util.MergeMaps(mvHashMap, mergeHashMap)

	// Return if no changes detected
	if len(combinedHashMap) == 0 {
		console.Warning("Local changes and branch \"%s\" are equivalent, aborting merge.", branchName)
		return nil
	}

	// Get temp dir for storing downloaded files
	tempDirPath := system.GetTempDir()

	// Download files from storage for:
	// - movable files
	// - mergable files
	//
	// NOTE: Downloaded files are already decompressed
	console.Info("Downloading required files...")
	console.Verbose("Temp directory: %s", tempDirPath)
	err = storage.DownloadMany(projectConfig, tempDirPath, combinedHashMap)
	if err != nil {
		return err
	}

	// Prompt user to confirm merge
	if confirm {
		console.Warning("Merge \"%s\" into \"%s\" (current)? (y/n)", branchToMerge.Name, currentBranch.Name)
		var answer string
		fmt.Scanln(&answer)

		if strings.ToLower(answer) != "y" {
			console.Info("Aborting...")

			// Delete temp dir
			console.Verbose("Deleting temp files from %s", tempDirPath)
			err = os.RemoveAll(tempDirPath)
			if err != nil {
				return err
			}

			return nil
		}
	}

	// Create empty base file for three-way merge
	baseFilePath := filepath.Join(tempDirPath, "empty")
	err = os.WriteFile(baseFilePath, []byte{}, 0644)
	if err != nil {
		return console.Error("Failed to create base file: %s", err)
	}

	// Move created files to project dir
	console.Verbose("Moving %d files to project...", len(mvHashMap))
	for path := range mvHashMap {
		dlPath := filepath.Join(tempDirPath, path)
		err = os.Rename(dlPath, path)
		if err != nil {
			return err
		}
	}

	console.Verbose("Merging %d files...", len(mergeHashMap))
	for path := range mergeHashMap {
		dlPath := filepath.Join(tempDirPath, path)
		cmd := exec.Command("git", "merge-file", path, baseFilePath, dlPath, "--union")
		err := cmd.Run()
		if err != nil {
			return console.Error("Failed to merge file \"%s\": %v", path, err)
		}
	}

	// Delete temp dir
	console.Verbose("Deleting temp files from %s", tempDirPath)
	err = os.RemoveAll(tempDirPath)
	if err != nil {
		return err
	}

	// Push if `push` flag provided (after user confirmation)
	// (This will also push local changes)
	if push {
		message := fmt.Sprintf("Merged %s into %s", branchToMerge.Name, currentBranch.Name)
		return Push(c, WithNoConfirm(), WithMessage(message))
	}

	return nil
}
