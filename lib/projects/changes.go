package projects

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/TwiN/go-color"
	"github.com/cespare/xxhash/v2"
	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/storj"
	"github.com/joshnies/qc-cli/lib/util"
	"github.com/joshnies/qc-cli/models"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

// Get file hash. Can be used to detect file changes.
// Uses XXH64 algorithm.
//
// @param path - File path
//
// Returns file hash.
func GetFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := xxhash.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Result of `projects.DetectFileChanges()`
type FileChangeDetectionResult struct {
	CreatedFilePaths  []string
	ModifiedFilePaths []string
	DeletedFilePaths  []string
	// Map of file path to hash
	HashMap map[string]string
}

// Detect file changes.
//
// @param oldHashMap - Hash map of remote commit
func DetectFileChanges(oldHashMap map[string]string) (FileChangeDetectionResult, error) {
	console.Info("Checking for changes...")

	// Get known file paths in current commit
	remainingPaths := maps.Keys(oldHashMap)

	createdFilePaths := []string{}
	modifiedFilePaths := []string{}
	newHashMap := make(map[string]string)
	fileInfoMap := make(map[string]os.FileInfo)

	// Walk project directory
	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() || filepath.Base(path) == constants.ProjectFileName {
			return nil
		}

		// Calculate file hash
		newHash, err := GetFileHash(path)
		if err != nil {
			return err
		}

		newHashMap[path] = newHash
		fileInfoMap[path] = info

		// Detect changes
		if oldHash, ok := oldHashMap[path]; ok {
			if oldHash != newHash {
				// File was modified
				modifiedFilePaths = append(modifiedFilePaths, path)
			}
		} else {
			// File is new
			createdFilePaths = append(createdFilePaths, path)
		}

		// Remove file path from remaining file paths
		remainingPaths = lo.Filter(remainingPaths, func(p string, _ int) bool {
			return p != path
		})

		return nil
	})
	if err != nil {
		return FileChangeDetectionResult{}, console.Error("Failed to detected changes: %v", err)
	}

	// Print result
	if len(createdFilePaths) > 0 {
		fmt.Println(color.InGreen(color.InBold("Created files:")))
		for _, fp := range createdFilePaths {
			fileInfo := fileInfoMap[fp]
			fmt.Printf(color.InGreen("  + %s (%s)\n"), fp, util.FormatBytesSize(fileInfo.Size()))
		}
	}
	if len(modifiedFilePaths) > 0 {
		console.Info(color.InBlue(color.InBold("Modified files:")))
		for _, fp := range modifiedFilePaths {
			fileInfo := fileInfoMap[fp]
			fmt.Printf(color.InBlue("  * %s (%s)\n"), fp, util.FormatBytesSize(fileInfo.Size()))
		}
	}
	if len(remainingPaths) > 0 {
		console.Info(color.InRed(color.InBold("Deleted files:")))
		for _, fp := range remainingPaths {
			fileInfo := fileInfoMap[fp]
			fmt.Printf(color.InRed("  - %s (%s)\n"), fp, util.FormatBytesSize(fileInfo.Size()))
		}
	}

	// Return result
	res := FileChangeDetectionResult{
		CreatedFilePaths:  createdFilePaths,
		ModifiedFilePaths: modifiedFilePaths,
		DeletedFilePaths:  remainingPaths,
		HashMap:           newHashMap,
	}

	return res, nil
}

// Reset all local changes.
// This will:
//
// - Delete all created files
//
// - Revert all modified files to their original state
//
// - Recreate all deleted files
//
// @param gc Global config
// @param confirm Whether to prompt user for confirmation before resetting
//
func ResetChanges(gc models.GlobalConfig, confirm bool) error {
	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current commit
	apiUrl := api.BuildURLf("projects/%s/commits/index/%d", projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
	commitRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return console.Error("Failed to get commit: %s", err)
	}

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&commit)
	if err != nil {
		return console.Error("Failed to parse commit: %s", err)
	}

	// Detect file changes
	fc, err := DetectFileChanges(commit.HashMap)
	if err != nil {
		return err
	}

	if len(fc.CreatedFilePaths) == 0 && len(fc.ModifiedFilePaths) == 0 && len(fc.DeletedFilePaths) == 0 {
		console.Info("No changes detected")
		return nil
	}

	// Prompt user for confirmation
	if confirm {
		console.Warning("You are about to reset all local changes. This will:")
		console.Warning("- Delete all created files")
		console.Warning("- Revert all modified files to their original state")
		console.Warning("- Recreate all deleted files")
		console.Warning("")
		console.Warning("Continue? (y/n)")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" {
			console.Info("Aborted")
			return nil
		}
	}

	// Delete all created files
	for _, path := range fc.CreatedFilePaths {
		err = os.Remove(path)
		if err != nil {
			return console.Error("Failed to delete file \"%s\": %s", path, err)
		}
	}

	// Build hash map for overridden files (modified + deleted)
	overrideHashMap := make(map[string]string)
	overrideFilePaths := append(fc.ModifiedFilePaths, fc.DeletedFilePaths...)
	for _, path := range overrideFilePaths {
		hash := commit.HashMap[path]
		overrideHashMap[path] = hash
	}

	// Download remote versions of modified and deleted files
	dataMap, err := storj.DownloadBulk(projectConfig.ProjectID, maps.Values(overrideHashMap))
	if err != nil {
		return console.Error("Failed to download files: %s", err)
	}

	// Write files to disk
	for hash, data := range dataMap {
		// Get file path from hash (reverse lookup)
		path := util.ReverseLookup(overrideHashMap, hash)

		if path == "" {
			return console.Error("Failed to find downloaded file with hash %s", hash)
		}

		// Write file
		err = ioutil.WriteFile(path, data, 0644)
		if err != nil {
			return console.Error("Failed to write file \"%s\" (hash %s): %s", path, hash, err)
		}
	}

	return nil
}
