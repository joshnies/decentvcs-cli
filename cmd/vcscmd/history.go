package vcscmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/TwiN/go-color"
	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/httpvalidation"
	"github.com/joshnies/decent/lib/vcs"
	"github.com/joshnies/decent/models"
	"github.com/urfave/cli/v2"
)

// Print commit history
func PrintHistory(c *cli.Context) error {
	auth.HasToken()

	// Parse args
	limit := c.Int("limit")
	if limit <= 0 {
		limit = 10
	}

	// Get project config
	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get commits up to limit
	httpClient := http.Client{}
	reqUrl := fmt.Sprintf("%s/projects/%s/commits?limit=%d", config.I.VCS.ServerHost, projectConfig.ProjectID, limit)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response
	var commits []models.CommitWithBranch
	err = json.NewDecoder(res.Body).Decode(&commits)
	if err != nil {
		return err
	}

	// Print commits
	for _, c := range commits {
		createdAt := time.Unix(c.CreatedAt, 0).Format(time.RFC1123)
		fmt.Printf("%s "+color.InCyan(color.InBold("[%s; #%d]"))+" %s\n", createdAt, c.Branch.Name, c.Index, c.Message)
	}

	return nil
}
