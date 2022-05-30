package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/TwiN/go-color"
	"github.com/joshnies/quanta/config"
	"github.com/joshnies/quanta/lib/auth"
	"github.com/joshnies/quanta/lib/console"
	"github.com/joshnies/quanta/lib/httpvalidation"
	"github.com/joshnies/quanta/lib/projects"
	"github.com/joshnies/quanta/lib/storage"
	"github.com/joshnies/quanta/lib/util"
	"github.com/joshnies/quanta/models"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/urfave/cli/v2"
	"github.com/xyproto/binary"
)

// Merge the specified branch into the current branch.
// User must be synced with remote first.
func Merge(c *cli.Context) error {
	gc := auth.Validate()

	// Extract args
	branchName := c.Args().Get(0)
	if branchName == "" {
		return console.Error("Please specify name of branch to merge")
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
		return console.Error("You are not synced with the remote. Please run `quanta pull`.")
	}

	// Get specified branch w/ commit
	reqUrl = fmt.Sprintf("%s/projects/%s/branches/%s?join_commit=true", config.I.API.Host, projectConfig.ProjectID, branchName)
	req, err = http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gc.Auth.AccessToken))
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

	// Detect local changes
	// TODO: Use user-provided project path if available
	fc, err := projects.DetectFileChanges(".", currentBranch.Commit.HashMap)
	if err != nil {
		return err
	}

	// Detect new files in branch to merge
	createdHashMap := make(map[string]string)
	for path, hash := range branchToMerge.Commit.HashMap {
		if _, ok := fc.HashMap[path]; !ok {
			createdHashMap[path] = hash
		}
	}

	// Get difference between local hash map and the hash map of the branch to merge
	modifiedHashMap := make(map[string]string)
	for path, hash := range fc.HashMap {
		newHash := branchToMerge.Commit.HashMap[path]
		if hash != newHash {
			modifiedHashMap[path] = newHash
		}
	}

	combinedHashMap := util.MergeMaps(createdHashMap, modifiedHashMap)

	// Return if no changes detected
	if len(combinedHashMap) == 0 {
		fmt.Println("No changes detected, nothing to merge.")
		return nil
	}

	// Create temp dir for storing downloaded files
	tempDirPath, err := os.MkdirTemp("", "quanta-merge-")
	if err != nil {
		return err
	}

	// Download created and modified files from storage
	// NOTE: Downloaded files are already decompressed
	console.Info("Downloading created & modified files...")
	console.Verbose("Temp directory: %s", tempDirPath)
	err = storage.DownloadMany(projectConfig.ProjectID, tempDirPath, combinedHashMap)
	if err != nil {
		return err
	}

	// Print changes to be merged.
	// For binary files, only show file name and size (compressed).
	// For text-based files, show file name and diff.
	diffMap := make(map[string][]diffmatchpatch.Diff)
	if len(modifiedHashMap) > 0 {
		console.Info(color.InBlue(color.InBold("Modified files:")))
		dmp := diffmatchpatch.New()
		for localPath := range modifiedHashMap {
			dlPath := filepath.Join(tempDirPath, localPath)
			isBinary, err := binary.File(dlPath)
			if err != nil {
				return err
			}

			localInfo, err := os.Stat(localPath)
			if err != nil {
				return err
			}

			localSize := localInfo.Size()

			dlInfo, err := os.Stat(dlPath)
			if err != nil {
				return err
			}

			dlSize := dlInfo.Size()
			dlSizeFormatted := util.FormatBytesSize(dlSize)

			if isBinary {
				// Print file name and size
				fmt.Printf(color.InBlue("%s (%s)\n"), localPath, dlSizeFormatted)
			} else {
				// Print file name and diff
				//
				// Ensure local file and downloaded file are not too big to read into memory
				if dlSize > config.I.MaxFileSizeForDiff {
					console.Warning("Merging version of file \"%s\" (%s) is too big to show diff, skipping", localPath, dlSizeFormatted)
					fmt.Printf(color.InBlue("%s (%s)\n"), localPath, dlSizeFormatted)
					continue
				}

				if localSize > config.I.MaxFileSizeForDiff {
					console.Warning("Local version of file \"%s\" (%s) is too big to show diff, skipping", localPath, dlSizeFormatted)
					fmt.Printf(color.InBlue("%s (%s)\n"), localPath, dlSizeFormatted)
					continue
				}

				// Read local file
				localFileBytes, err := ioutil.ReadFile(localPath)
				if err != nil {
					return err
				}
				localFileStr := string(localFileBytes)

				// Read downloaded (merging) file
				dlFileBytes, err := ioutil.ReadFile(dlPath)
				if err != nil {
					return err
				}
				dlFileStr := string(dlFileBytes)

				// Create and print diff
				diffs := dmp.DiffMain(localFileStr, dlFileStr, true)
				fmt.Printf(color.InBlue("%s (%s)\n"), localPath, dlSizeFormatted)
				fmt.Println(dmp.DiffPrettyText(diffs))
				fmt.Println()

				diffMap[localPath] = diffs
			}
		}
	}

	// TODO: Prompt user to confirm merge

	// TODO: Move created files to project dir

	// TODO: Merge modified files

	// TODO: Delete temp dir

	// TODO: Push if `push` flag provided (after user confirmation)

	return nil
}
