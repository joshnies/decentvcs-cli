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

type FileChangeDetectionResult struct {
	Changes                   []models.FileChange
	State                     map[string]models.CommitState
	PathsToUpdateHostCommitId []string
}

// Detect file changes.
func DetectFileChanges(state map[string]models.CommitState) (FileChangeDetectionResult, error) {
	// Get currently-known file paths
	remainingPaths := maps.Keys(state)

	var changes []models.FileChange
	newState := make(map[string]models.CommitState)
	pathsToUpdateHostCommitId := []string{}

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

		oldState, isInOldState := state[path]
		if !isInOldState {
			// File is new
			pathsToUpdateHostCommitId = append(pathsToUpdateHostCommitId, path)
		}

		newState[path] = models.CommitState{
			Hash:         newHash,
			HostCommitId: oldState.HostCommitId,
		}

		// Detect changes
		// If host commit is unknown, then the file is new since it's never been uploaded to storage
		if isInOldState {
			if oldState.Hash != newHash {
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
		return FileChangeDetectionResult{}, console.Error("Failed to detected changes: %v", err)
	}

	// Add changes for deleted files
	for _, path := range remainingPaths {
		change := models.FileChange{
			Path: path,
			Type: models.FileWasDeleted,
		}
		changes = append(changes, change)
	}

	// Return result
	res := FileChangeDetectionResult{
		Changes:                   changes,
		State:                     newState,
		PathsToUpdateHostCommitId: pathsToUpdateHostCommitId,
	}

	return res, nil
}
