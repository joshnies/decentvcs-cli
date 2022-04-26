package lib

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/joshnies/qc-cli/models"
)

// Write project file.
func CreateProjectFile(path string, data models.ProjectFileData) error {
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(path, ".qc"), json, os.ModePerm)
}

// Detect file changes.
//
// Returns a list of paths to changed files.
func DetectFileChanges() ([]string, error) {
	history := map[string]int64{}

	// Read project history file
	historyFile, err := os.Open(".qchistory")
	if os.IsNotExist(err) {
		// Create project history file
		_, createErr := os.Create(".qchistory")
		if createErr != nil {
			return nil, createErr
		}
	} else {
		// Return error if not a "file not found" error
		return nil, err
	}

	if err == nil {
		// Decode JSON
		err = json.NewDecoder(historyFile).Decode(&history)
		if err != nil {
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
