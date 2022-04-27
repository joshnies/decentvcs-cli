package lib

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/models"
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
// Returns history data.
func ReadHistory() (HistoryData, error) {
	historyFile, err := os.Open(constants.HistoryFileName)
	if os.IsNotExist(err) {
		// Write empty history file
		historyJson, _ := json.Marshal(HistoryData{})
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
	var history HistoryData
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

// Detect file changes.
//
// Returns:
//
// - list of paths to changed files
//
// - list of modification times in Unix seconds
//
// - error
func DetectFileChanges() ([]string, []int64, error) {
	// Read project history file
	history, err := ReadHistory()
	if err != nil {
		return nil, nil, err
	}

	changedFiles := []string{}
	modTimes := []int64{}

	// Walk project directory
	filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		currentModTime := info.ModTime().Unix()

		// Check if file has changed or doesnt exist in remote yet
		// TODO: Check file size
		if lastModTime, ok := history[path]; ok {
			if currentModTime > lastModTime {
				changedFiles = append(changedFiles, path)
				modTimes = append(modTimes, currentModTime)
				return nil
			}
		} else {
			changedFiles = append(changedFiles, path)
			modTimes = append(modTimes, currentModTime)
		}

		return nil
	})

	return changedFiles, modTimes, nil
}
