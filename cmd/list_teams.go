package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/decentvcs/cli/models"
	"github.com/urfave/cli/v2"
)

// List all teams for the user.
func ListTeams(c *cli.Context) error {
	// Get all teams for the user
	var httpClient http.Client
	reqUrl := fmt.Sprintf("%s/teams?mine=true", config.I.VCS.ServerHost)
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Parse response body
	var teams []models.Team
	if err = json.NewDecoder(res.Body).Decode(&teams); err != nil {
		return err
	}

	// Print teams
	console.Info("Your teams:")
	for _, team := range teams {
		console.Info("  " + team.Name)
	}

	return nil
}
