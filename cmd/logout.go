package cmd

import (
	"net/http"
	"os"

	"github.com/joshnies/dvcs/config"
	"github.com/joshnies/dvcs/constants"
	"github.com/joshnies/dvcs/lib/auth"
	"github.com/joshnies/dvcs/lib/console"
	"github.com/joshnies/dvcs/lib/httpvalidation"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log out.
func LogOut(c *cli.Context) error {
	auth.HasToken()

	// Revoke the session
	httpClient := http.Client{}
	req, _ := http.NewRequest("DELETE", config.I.VCS.ServerHost+"/session", nil)
	req.Header.Set("X-Session-Token", config.I.Auth.SessionToken)
	res, err := httpClient.Do(req)
	if err != nil {
		console.Warning("Could not revoke the session: %s", err.Error())
		return err
	}
	if err = httpvalidation.ValidateResponse(res); err != nil {
		console.Warning("Could not revoke the session: %s", err.Error())
		return err
	}

	// Clear auth data
	config.I.Auth = config.AuthConfig{}

	// Save global config file
	cYaml, err := yaml.Marshal(config.I)
	if err != nil {
		console.Verbose("Error while encoding auth config as JSON: %s", err)
		return console.Error(constants.ErrInternal)
	}

	err = os.WriteFile(config.GetConfigPath(), cYaml, 0644)
	if err != nil {
		return err
	}

	console.Info("Logged out")
	return nil
}
