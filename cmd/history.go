package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Print commit history
func PrintHistory(c *cli.Context) error {
	gc := auth.Validate()

	// Parse args
	limit := c.Int("limit")
	if limit <= 0 {
		limit = 10
	}

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get commits up to limit
	apiUrl := api.BuildURLf("projects/%s/commits?limit=%d", projectConfig.ProjectID, limit)
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse response
	var commits []models.CommitWithBranch
	err = json.NewDecoder(res.Body).Decode(&commits)
	if err != nil {
		return err
	}

	// Print commits
	for _, c := range commits {
		// TODO: Colorize
		createdAt := time.Unix(c.CreatedAt, 0).Format(time.RFC1123)
		fmt.Printf("%s [%s; #%d] %s\n", createdAt, c.Branch.Name, c.Index, c.Message)
	}

	return nil
}
