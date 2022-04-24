package projects

import (
	"encoding/json"
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
