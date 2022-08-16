package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/TwiN/go-color"
	"github.com/joshnies/dvcs/config"
	"github.com/joshnies/dvcs/constants"
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/httpvalidation"
	"github.com/joshnies/dvcs/lib/vcs"
	"github.com/joshnies/dvcs/models"
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
	reqUrl := fmt.Sprintf("%s/projects/%s/commits?limit=%d", config.I.VCS.ServerHost, projectConfig.ProjectSlug, limit)
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
		createdAt := c.CreatedAt.Format(time.RFC1123)
		fmt.Printf("%s "+color.InCyan(color.InBold("[%s; #%d]"))+" %s\n", createdAt, c.Branch.Name, c.Index, c.Message)
	}

	return nil
}
