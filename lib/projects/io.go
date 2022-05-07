package projects

import (
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/models"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

// Get file hash. Can be used to detect file changes.
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

// Detect file changes.
//
// Returns:
//
// - list of FileChange objects.
//
// - regenerated hash map.
//
// - error.
func DetectFileChanges(hashMap map[string]string) ([]models.FileChange, map[string]string, error) {
	// Get currently-known file paths
	remainingPaths := maps.Keys(hashMap)

	var changes []models.FileChange
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
		if oldHash, ok := hashMap[path]; ok {
			if oldHash != newHash {
				// File was modified
				changes = append(changes, models.FileChange{
					Path: path,
					Type: models.FileWasModified,
				})
			}
		} else {
			// File is new
			changes = append(changes, models.FileChange{
				Path: path,
				Type: models.FileWasCreated,
			})
		}

		// Remove file path from remaining file paths
		remainingPaths = lo.Filter(remainingPaths, func(p string, _ int) bool {
			return p != path
		})

		return nil
	})
	if err != nil {
		return nil, nil, console.Error("Failed to detected changes: %v", err)
	}

	// Add changes for deleted files
	for _, path := range remainingPaths {
		change := models.FileChange{
			Path: path,
			Type: models.FileWasDeleted,
		}
		changes = append(changes, change)
	}

	return changes, newHashMap, nil
}
