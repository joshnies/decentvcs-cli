package vcscmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/joshnies/dvcs/config"
	"github.com/joshnies/dvcs/constants"
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/console"
	"github.com/joshnies/dvcs/lib/httpvalidation"
	"github.com/joshnies/dvcs/lib/system"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/urfave/cli/v2"
)

// Unlock one or many files, allowing other users to edit them again.
// Specific to a branch.
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

	force := c.Bool("force")

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

	var queryParam string
	if force {
		queryParam = "?force=true"
	}

	reqURL := fmt.Sprintf(config.I.VCS.ServerHost+"/projects/%s/branches/%s/locks%s", projectConfig.ProjectSlug, projectConfig.CurrentBranchName, queryParam)
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
