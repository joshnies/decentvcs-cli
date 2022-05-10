package main

import (
	"log"
	"os"

	"github.com/joshnies/qc-cli/cmd"
	"github.com/joshnies/qc-cli/config"
	"github.com/urfave/cli/v2"
)

func main() {
	// Initialize config
	config.InitConfig()

	// Initialize CLI app
	app := &cli.App{
		Name:    "qc",
		Usage:   "Quanta Control CLI",
		Version: "0.0.1",
		Commands: []*cli.Command{
			{
				Name:   "login",
				Usage:  "Authenticate with Quanta Control (required to use other commands)",
				Action: cmd.LogIn,
			},
			{
				Name:   "logout",
				Usage:  "Log out of Quanta Control",
				Action: cmd.LogOut,
			},
			{
				Name:   "auth",
				Usage:  "Print current authentication state",
				Action: cmd.PrintAuthState,
			},
			{
				Name:    "init",
				Usage:   "Initialize a new project",
				Aliases: []string{"i"},
				Action:  cmd.Init,
			},
			{
				Name:    "push",
				Usage:   "Push local changes to remote",
				Aliases: []string{"p"},
				Action:  cmd.Push,
			},
			{
				Name:    "sync",
				Usage:   "Sync to commit, downloading changes from remote",
				Aliases: []string{"to", "t"},
				Action:  cmd.Sync,
			},
			{
				Name:    "changes",
				Usage:   "Print current changes",
				Aliases: []string{"c"},
				Action:  cmd.GetChanges,
			},
			{
				Name:    "revert",
				Usage:   "Reset all current changes and sync to last commit",
				Aliases: []string{"r"},
				Action:  cmd.Revert,
			},
			{
				Name:    "status",
				Usage:   "Print local project status",
				Aliases: []string{"s"},
				Action:  cmd.PrintStatus,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
