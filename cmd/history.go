package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/TwiN/go-color"
	"github.com/joshnies/quanta-cli/config"
	"github.com/joshnies/quanta-cli/lib/api"
	"github.com/joshnies/quanta-cli/lib/auth"
	"github.com/joshnies/quanta-cli/lib/httpw"
	"github.com/joshnies/quanta-cli/models"
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
		createdAt := time.Unix(c.CreatedAt, 0).Format(time.RFC1123)
		fmt.Printf("%s "+color.InCyan(color.InBold("[%s; #%d]"))+" %s\n", createdAt, c.Branch.Name, c.Index, c.Message)
	}

	return nil
}
