package globalcmd

import (
	"io/ioutil"

	"github.com/joshnies/decent/config"
	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/auth"
	"github.com/joshnies/decent/lib/console"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Log out.
func LogOut(c *cli.Context) error {
	auth.Validate()

	// Clear auth data
	config.I.Auth = config.AuthConfig{}

	// Save global config file
	cYaml, err := yaml.Marshal(config.I)
	if err != nil {
		console.Verbose("Error while encoding auth data as JSON: %s", err)
		return console.Error(constants.ErrMsgInternal)
	}

	err = ioutil.WriteFile(config.GetConfigPath(), cYaml, 0644)
	if err != nil {
		return err
	}

	console.Info("Logged out")
	return nil
}
