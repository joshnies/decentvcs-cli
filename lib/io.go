package lib

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/models"
)

// Write project file.
func WriteProjectConfig(path string, data models.ProjectFileData) error {
	configPath := filepath.Join(path, constants.ProjectFileName)

	// Read existing project file (if it exists)
	jsonFile, err := os.Open(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer jsonFile.Close()

	// Decode existing data JSON
	var existingData models.ProjectFileData
	if jsonFile != nil {
		err = json.NewDecoder(jsonFile).Decode(&existingData)
		if err != nil {
			// TODO: Improve this error
			return err
		}
	}

	// If existing data exists, merge it with new data
	mergedData := models.MergeProjectFileData(existingData, data)

	// Write
	json, err := json.MarshalIndent(mergedData, "", "  ")
	if err != nil {
		// TODO: Improve this error
		return err
	}

	return ioutil.WriteFile(configPath, json, os.ModePerm)
}

// Detect file changes.
//
// Returns a list of paths to changed files.
func DetectFileChanges() ([]string, error) {
	history := map[string]int64{}

	// Read project history file
	historyFile, err := os.Open(constants.HistoryFileName)
	if os.IsNotExist(err) {
		// Write empty history file
		historyJson, err := json.Marshal(map[string]int64{})
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(constants.HistoryFileName, historyJson, os.ModePerm)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		// Return error if not a "file not found" error
		fmt.Println("Error reading history file")
		return nil, err
	} else {
		// Decode JSON
		err = json.NewDecoder(historyFile).Decode(&history)
		if err != nil {
			fmt.Println("Error reading history file")
			return nil, err
		}
	}

	changedFiles := []string{}

	// Walk project directory
	filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file has changed or doesnt exist in remote yet
		// TODO: Check file size
		if lastModTime, ok := history[path]; ok {
			if info.ModTime().Unix() > lastModTime {
				changedFiles = append(changedFiles, path)
				return nil
			}
		} else {
			changedFiles = append(changedFiles, path)
		}

		return nil
	})

	return changedFiles, nil
}
