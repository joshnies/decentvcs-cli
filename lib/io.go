package lib

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/models"
	"github.com/samber/lo"
)

// History file data
type HistoryData map[string]int64

// Write project file.
func WriteProjectConfig(path string, data models.ProjectConfig) (models.ProjectConfig, error) {
	configPath := filepath.Join(path, constants.ProjectFileName)

	// Read existing project file (if it exists)
	jsonFile, err := os.Open(configPath)
	if err != nil && !os.IsNotExist(err) {
		return models.ProjectConfig{}, err
	}
	defer jsonFile.Close()

	// Decode existing data JSON
	var existingData models.ProjectConfig
	if jsonFile != nil {
		err = json.NewDecoder(jsonFile).Decode(&existingData)
		if err != nil {
			// TODO: Improve this error
			return models.ProjectConfig{}, err
		}
	}

	// If existing data exists, merge it with new data
	mergedData := models.MergeProjectConfigs(existingData, data)

	// Write
	json, err := json.MarshalIndent(mergedData, "", "  ")
	if err != nil {
		// TODO: Improve this error
		return models.ProjectConfig{}, err
	}

	err = ioutil.WriteFile(configPath, json, os.ModePerm)
	return mergedData, err
}

// Read project history file.
//
// Returns list history entries.
func ReadHistory() ([]models.HistoryEntry, error) {
	historyFile, err := os.Open(constants.HistoryFileName)
	if os.IsNotExist(err) {
		// Write empty history file
		historyJson, _ := json.Marshal([]HistoryData{})
		err = ioutil.WriteFile(constants.HistoryFileName, historyJson, os.ModePerm)
		if err != nil {
			return nil, Log(LogOptions{
				Level:       Error,
				Str:         "Failed to write history file",
				VerboseStr:  "%v",
				VerboseVars: []interface{}{err},
			})
		}
	} else if err != nil {
		// Return error if not a "file not found" error
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to read history file. It may be invalid/corrupt, in which case you should delete it from your local system.",
			VerboseStr:  "%v",
			VerboseVars: []interface{}{err},
		})
	}

	// Decode JSON
	var history []models.HistoryEntry
	err = json.NewDecoder(historyFile).Decode(&history)
	if err != nil {
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Error reading history file",
			VerboseStr:  "%v",
			VerboseVars: []interface{}{err},
		})
	}

	return history, nil
}

// Get file SHA1 hash. Can be used to detect file changes.
// TODO: Use xxhash instead of SHA1
func GetFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
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
// - list of history entries.
//
// - error.
func DetectFileChanges() ([]models.FileChange, []models.HistoryEntry, error) {
	// Read project history file
	currentHistory, err := ReadHistory()
	if err != nil {
		return nil, nil, err
	}

	// Get currently-known file paths
	knownPaths := lo.Map(currentHistory, func(entry models.HistoryEntry, _ int) string {
		return entry.Path
	})

	var changes []models.FileChange
	var history []models.HistoryEntry

	// Walk project directory
	err = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate file hash
		newHash, err := GetFileHash(path)
		if err != nil {
			return err
		}

		// Get existing history entry
		existingHistoryEntry, ok := lo.Find(currentHistory, func(entry models.HistoryEntry) bool {
			return entry.Path == path
		})

		// Create new history entry
		newHistoryEntry := models.HistoryEntry{
			Path: path,
			Hash: newHash,
		}

		history = append(history, newHistoryEntry)

		var change *models.FileChange

		if ok {
			if newHash != existingHistoryEntry.Hash {
				// File was modified
				change = &models.FileChange{
					Path:            path,
					Type:            models.FileWasModified,
					NewHistoryEntry: newHistoryEntry,
				}
			}
		} else {
			// File is new
			change = &models.FileChange{
				Path:            path,
				Type:            models.FileWasCreated,
				NewHistoryEntry: newHistoryEntry,
			}
		}

		if change != nil {
			// Add change to list
			changes = append(changes, *change)

			// Remove file path from remaining file paths
			knownPaths = lo.Filter(knownPaths, func(p string, _ int) bool {
				return p != path
			})
		}

		return nil
	})
	if err != nil {
		return nil, nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to detect file changes",
			VerboseStr:  "%v",
			VerboseVars: []interface{}{err},
		})
	}

	// Add changes for deleted files (remaining items in knownPaths)
	for _, path := range knownPaths {
		change := models.FileChange{
			Path: path,
			Type: models.FileWasDeleted,
		}
		changes = append(changes, change)
	}

	return changes, history, nil
}
