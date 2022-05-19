package projects

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TwiN/go-color"
	"github.com/cespare/xxhash/v2"
	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/storage"
	"github.com/joshnies/qc/lib/util"
	"github.com/joshnies/qc/models"
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
func DetectFileChanges(projectPath string, oldHashMap map[string]string) (FileChangeDetectionResult, error) {
	console.Info("Checking for changes...")

	// Get known file paths in current commit
	remainingPaths := maps.Keys(oldHashMap)

	createdFilePaths := []string{}
	modifiedFilePaths := []string{}
	newHashMap := make(map[string]string)
	fileInfoMap := make(map[string]os.FileInfo)

	createdFileSizeTotal := int64(0)
	modifiedFileSizeTotal := int64(0)
	deletedFileSizeTotal := int64(0)

	// Read .qcignore file
	qcignorePath := filepath.Join(projectPath, constants.IgnoreFileName)
	qcignoreFile, err := os.Open(qcignorePath)
	if err != nil && !os.IsNotExist(err) {
		return FileChangeDetectionResult{}, err
	}
	defer qcignoreFile.Close()

	// Read .qcignore file
	ignoredFilePatterns := []string{}
	scanner := bufio.NewScanner(qcignoreFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ignoredFilePatterns = append(ignoredFilePatterns, line)
		}
	}

	// Walk project directory
	err = filepath.Walk(projectPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and QC project file (`.qc`)
		if info.IsDir() || filepath.Base(path) == constants.ProjectFileName {
			return nil
		}

		// Skip hidden files
		for _, pattern := range ignoredFilePatterns {
			matched, err := regexp.Match(pattern, []byte(path))
			if err != nil {
				return err
			}
			if matched {
				console.Verbose("Ignoring file \"%s\"", path)
				return nil
			}
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
			fileSize := fileInfo.Size()
			createdFileSizeTotal += fileSize
			fmt.Printf(color.InGreen("  + %s (%s)\n"), fp, util.FormatBytesSize(fileSize))
		}

		fmt.Printf(color.InGreen("  Total: %s\n"), util.FormatBytesSize(createdFileSizeTotal))
	}
	if len(modifiedFilePaths) > 0 {
		console.Info(color.InBlue(color.InBold("Modified files:")))
		for _, fp := range modifiedFilePaths {
			fileInfo := fileInfoMap[fp]
			fileSize := fileInfo.Size()
			modifiedFileSizeTotal += fileSize
			fmt.Printf(color.InBlue("  * %s (%s)\n"), fp, util.FormatBytesSize(fileSize))
		}

		console.Info(color.InBlue("  Total: %s\n"), util.FormatBytesSize(modifiedFileSizeTotal))
	}
	if len(remainingPaths) > 0 {
		console.Info(color.InRed(color.InBold("Deleted files:")))
		for _, fp := range remainingPaths {
			if fileInfo, ok := fileInfoMap[fp]; ok {
				fileSize := fileInfo.Size()
				deletedFileSizeTotal += fileSize
				fmt.Printf(color.InRed("  - %s (%s)\n"), fp, util.FormatBytesSize(fileSize))
			} else {
				fmt.Printf(color.InRed("  - %s\n"), fp)
			}
		}

		console.Info(color.InRed("  Total: %s\n"), util.FormatBytesSize(deletedFileSizeTotal))
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
	// TODO: Use user-provided project path if available
	fc, err := DetectFileChanges(".", commit.HashMap)
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
	err = storage.DownloadMany(projectConfig.ProjectID, overrideHashMap)
	if err != nil {
		return console.Error("Failed to download files: %s", err)
	}

	return nil
}
