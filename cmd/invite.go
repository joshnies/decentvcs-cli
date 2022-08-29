package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/decentvcs/cli/config"
	"github.com/decentvcs/cli/constants"
	"github.com/decentvcs/cli/lib/auth"
	"github.com/decentvcs/cli/lib/console"
	"github.com/decentvcs/cli/lib/vcs"
	"github.com/urfave/cli/v2"
)

// Invite users to a project via email.
// Existing users simply obtain a new permission for the project if they don't have one already, and are notified via
// email.
//
// Emails are separated by spaces.
func Invite(c *cli.Context) error {
	auth.HasToken()

	projectConfig, err := vcs.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get emails from args
	emails := c.Args().Slice()

	// Validate emails
	for _, email := range emails {
		if _, err := mail.ParseAddress(email); err != nil {
			return console.Error("Invalid email: %s", email)
		}
	}

	teamName := strings.Split(projectConfig.ProjectSlug, "/")[0]

	// Invite users
	httpClient := &http.Client{}
	body, _ := json.Marshal(map[string]any{
		"emails": emails,
	})
	reqUrl := fmt.Sprintf("%s/teams/%s/invite", config.I.VCS.ServerHost, teamName)
	req, _ := http.NewRequest("POST", reqUrl, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(constants.SessionTokenHeader, config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode > 201 {
		return console.Error("Error inviting users: received status code %d", res.StatusCode)
	}

	console.Info("Invited %d users to the %s team", len(emails), teamName)
	return nil
}