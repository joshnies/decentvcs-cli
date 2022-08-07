package vcscmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/dvcs/config"
	"github.com/joshnies/dvcs/constants"
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/console"
	"github.com/joshnies/dvcs/lib/httpvalidation"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/urfave/cli/v2"
)

// Lock one or many files from edits by other users.
// File(s) must exist in remote.
// Specific to a branch.
func Lock(c *cli.Context) error {
	auth.HasToken()

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get file paths from args
	paths := c.Args().Slice()
	if len(paths) == 0 {
		return console.Error("Please specify at least one file path to lock")
	}

	// Lock files on the server
	httpClient := http.Client{}
	bodyData := map[string]interface{}{
		"paths": paths,
	}
	body, _ := json.Marshal(bodyData)
	reqURL := fmt.Sprintf(config.I.VCS.ServerHost+"/projects/%s/branches/%s/locks", projectConfig.ProjectSlug, projectConfig.CurrentBranchName)
	req, _ := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		console.ErrorPrint("Could not lock files")
		return console.Error("%v", err)
	} else if err := httpvalidation.ValidateResponse(res); err != nil {
		return console.Error("%v", err)
	}
	defer res.Body.Close()

	console.Success("Locked %d files, they're all yours!", len(paths))
	return nil
}
