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
	// Read project history file
	historyFile, err := os.Open(".qchistory")
	if err != nil {
		return nil, err
	}

	// Decode JSON
	history := map[string]int64{}
	err = json.NewDecoder(historyFile).Decode(&history)
	if err != nil {
		return nil, err
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

		// Check if file has changed
		// TODO: Check file size
		if lastModTime, ok := history[path]; ok {
			if info.ModTime().Unix() > lastModTime {
				changedFiles = append(changedFiles, path)
				return nil
			}
		}

		return nil
	})

	return changedFiles, nil
}
