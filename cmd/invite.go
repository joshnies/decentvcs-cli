package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/httpvalidation"
	"github.com/urfave/cli/v2"
)

// Invite users to a project via email.
// Existing users simply obtain a new permission for the project if they don't have one already, and are notified via
// email.
//
// Emails are separated by spaces.
func Invite(c *cli.Context) error {
	auth.HasToken()

	// Get emails from args
	emails := c.Args().Slice()

	// Validate emails
	for _, email := range emails {
		if _, err := mail.ParseAddress(email); err != nil {
			return console.Error("Invalid email: %s", email)
		}
	}

	teamName := c.String("team")

	// Get team
	httpClient := &http.Client{}
	reqUrl := fmt.Sprintf("%s/teams/%s", config.I.VCS.ServerHost, teamName)
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to get team: %s", err.Error())
	}
	if res.StatusCode == 404 {
		return console.Error("Team \"%s\" not found", teamName)
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	// Invite users
	body, _ := json.Marshal(map[string]any{
		"emails": emails,
	})
	reqUrl = fmt.Sprintf("%s/%s/invite", config.I.VCS.ServerHost, teamName)
	req, _ = http.NewRequest("POST", reqUrl, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err = httpClient.Do(req)
	if err != nil {
		return console.Error("Failed to invite users: %s", err.Error())
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		return err
	}

	console.Info("Invited %d users to the %s team", len(emails), teamName)
	return nil
}
