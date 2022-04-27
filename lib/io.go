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
// Returns a list of history entries.
func DetectFileChanges() ([]models.HistoryEntry, error) {
	// Read project history file
	currentHistory, err := ReadHistory()
	if err != nil {
		return nil, err
	}

	var newHistory []models.HistoryEntry

	// Walk project directory
	err = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		currentHash, err := GetFileHash(path)
		if err != nil {
			return err
		}

		// Check if file has been created, modified or deleted
		historyEntry, ok := lo.Find(currentHistory, func(entry models.HistoryEntry) bool {
			return entry.Path == path
		})

		if ok {
			// Check if file has been modified
			if currentHash != historyEntry.Hash {
				hash, err := GetFileHash(path)
				if err != nil {
					return err
				}

				newHistory = append(newHistory, models.HistoryEntry{
					Path: path,
					Hash: hash,
				})
			}
		} else {
			// File is new
			hash, err := GetFileHash(path)
			if err != nil {
				return err
			}

			newHistory = append(newHistory, models.HistoryEntry{
				Path: path,
				Hash: hash,
			})
		}

		return nil
	})
	if err != nil {
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to detect file changes",
			VerboseStr:  "%v",
			VerboseVars: []interface{}{err},
		})
	}

	return newHistory, nil
}
