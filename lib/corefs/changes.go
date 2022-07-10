package corefs

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TwiN/go-color"
	"github.com/cespare/xxhash/v2"
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/storage"
	"github.com/joshnies/decent/lib/util"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
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

// Calculate hash map for all local files.
// Ignores files that match ignore patterns.
//
// @param rootPath - Root directory path for where to start the calculation.
//
// @returns Map of file paths to hashes.
//
func CalculateHashes(rootPath string) (map[string]string, error) {
	console.Verbose("Calculating hashes...")

	// Get known file paths in current commit
	hashMap := make(map[string]string)

	// Read ignore file
	ignoreFilePath := filepath.Join(rootPath, constants.IgnoreFileName)
	ignoreFile, err := os.Open(ignoreFilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	defer ignoreFile.Close()

	// Read ignore file
	ignoredFilePatterns := []string{}
	scanner := bufio.NewScanner(ignoreFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ignoredFilePatterns = append(ignoredFilePatterns, line)
		}
	}

	// Walk project directory
	err = filepath.Walk(rootPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and project file
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

		hashMap[path] = newHash
		return nil
	})
	if err != nil {
		return nil, console.Error("Failed to calculate hashes: %v", err)
	}

	return hashMap, nil
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
// @param currentHashMap - Hash map of current commit fetched from remote
//
func DetectFileChanges(currentHashMap map[string]string) (FileChangeDetectionResult, error) {
	console.Info("Checking for changes...")

	// Get known file paths in current commit
	remainingPaths := maps.Keys(currentHashMap)

	createdFilePaths := []string{}
	modifiedFilePaths := []string{}
	newHashMap := make(map[string]string)
	fileInfoMap := make(map[string]os.FileInfo)

	createdFileSizeTotal := int64(0)
	modifiedFileSizeTotal := int64(0)

	// Get project config file path
	projectConfigPath, err := vcs.GetProjectConfigPath()
	if err != nil {
		return FileChangeDetectionResult{}, err
	}

	projectPath := filepath.Dir(projectConfigPath)

	// Get ignore file patterns
	ignoredFilePatterns, err := vcs.GetIgnoredFilePatterns()
	if err != nil {
		return FileChangeDetectionResult{}, err
	}

	// Walk project directory
	err = filepath.Walk(projectPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and project file
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
		if oldHash, ok := currentHashMap[path]; ok {
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
			relFilePath, _ := filepath.Rel(projectPath, fp)
			fmt.Printf(color.InGreen("  + %s (%s)\n"), relFilePath, util.FormatBytesSize(fileSize))
		}

		fmt.Printf(color.InGreen("  Total: %s\n"), util.FormatBytesSize(createdFileSizeTotal))
	}
	if len(modifiedFilePaths) > 0 {
		console.Info(color.InBlue(color.InBold("Modified files:")))
		for _, fp := range modifiedFilePaths {
			fileInfo := fileInfoMap[fp]
			fileSize := fileInfo.Size()
			modifiedFileSizeTotal += fileSize
			relFilePath, _ := filepath.Rel(projectPath, fp)
			fmt.Printf(color.InBlue("  * %s (%s)\n"), relFilePath, util.FormatBytesSize(fileSize))
		}

		console.Info(color.InBlue("  Total: %s\n"), util.FormatBytesSize(modifiedFileSizeTotal))
	}
	if len(remainingPaths) > 0 {
		console.Info(color.InRed(color.InBold("Deleted files:")))
		for _, fp := range remainingPaths {
			relFilePath, _ := filepath.Rel(projectPath, fp)
			fmt.Printf(color.InRed("  - %s\n"), relFilePath)
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
// @param confirm Whether to prompt user for confirmation before resetting
//
func ResetChanges(confirm bool) error {
	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get current commit
	httpClient := http.Client{}
	url := fmt.Sprintf("%s/projects/%s/commits/index/%d", config.I.VCS.ServerHost, projectConfig.ProjectID, projectConfig.CurrentCommitIndex)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to get commit: %s", err)
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return console.Error("Failed to get commit: %s", err)
	}
	defer res.Body.Close()

	// Parse commit
	var commit models.Commit
	err = json.NewDecoder(res.Body).Decode(&commit)
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
	err = storage.DownloadMany(projectConfig.ProjectID, ".", overrideHashMap)
	if err != nil {
		return console.Error("Failed to download files: %s", err)
	}

	return nil
}
