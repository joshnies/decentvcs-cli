package projects

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/TwiN/go-color"
	"github.com/cespare/xxhash/v2"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

// Get file hash. Can be used to detect file changes.
// Uses XXH64 algorithm.
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

type FileChangeDetectionResult struct {
	CreatedFilePaths  []string
	ModifiedFilePaths []string
	DeletedFilePaths  []string
	// Map of file path to hash
	HashMap map[string]string
}

// Detect file changes.
func DetectFileChanges(oldHashMap map[string]string) (FileChangeDetectionResult, error) {
	console.Info("Checking for changes...")

	// Get known file paths in current commit
	remainingPaths := maps.Keys(oldHashMap)

	createdFilePaths := []string{}
	modifiedFilePaths := []string{}
	newHashMap := make(map[string]string)

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
			fmt.Printf(color.InGreen("  + %s\n"), fp)
		}
	}
	if len(modifiedFilePaths) > 0 {
		console.Info(color.InBlue(color.InBold("Modified files:")))
		for _, fp := range modifiedFilePaths {
			fmt.Printf(color.InBlue("  * %s\n"), fp)
		}
	}
	if len(remainingPaths) > 0 {
		console.Info(color.InRed(color.InBold("Deleted files:")))
		for _, fp := range remainingPaths {
			fmt.Printf(color.InRed("  - %s\n"), fp)
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
