package vcscmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/system"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/urfave/cli/v2"
)

// Unlock one or many files, allowing other users to edit them again.
// Specific to a branch.
// TODO: Add support for unlocking remote-only files
func Unlock(c *cli.Context) error {
	auth.HasToken()

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get file paths from args
	originPaths := c.Args().Slice()
	if len(originPaths) == 0 {
		return console.Error("Please specify at least one file or directory to unlock")
	}

	var paths []string
	for _, path := range originPaths {
		// Check if directory
		stat, err := os.Stat(path)
		if err != nil {
			return console.Error("Could not stat file \"%s\", it probably doesn't exist on your local machine", path)
		}

		if stat.IsDir() {
			// Is directory, get all files in it
			newPaths, err := system.ListFiles(path)
			if err != nil {
				return console.Error("Could not list files in directory \"%s\"", path)
			}
			paths = append(paths, newPaths...)
		} else {
			// Is file
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return console.Error("No files found in the given directories")
	}

	// Unlock files on the server
	httpClient := http.Client{}
	bodyData := map[string]interface{}{
		"paths": paths,
	}
	body, _ := json.Marshal(bodyData)
	reqURL := fmt.Sprintf(config.I.VCS.ServerHost+"/projects/%s/branches/%s/locks", projectConfig.ProjectID, projectConfig.CurrentBranchID)
	req, _ := http.NewRequest("DELETE", reqURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil || httpvalidation.ValidateResponse(res) != nil {
		return console.Error("Could not unlock files")
	}
	defer res.Body.Close()

	console.Success("Unlocked %d files", len(paths))
	return nil
}
