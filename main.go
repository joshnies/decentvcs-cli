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
				Name:    "auth",
				Aliases: []string{"a"},
				Usage:   "Print current authentication state",
				Action:  cmd.PrintAuthState,
			},
			{
				Name:    "init",
				Usage:   "Initialize a new project",
				Aliases: []string{"i"},
				Action:  cmd.Init,
			},
			{
				Name:    "changes",
				Usage:   "Print current changes",
				Aliases: []string{"c"},
				Action:  cmd.GetChanges,
			},
			{
				Name:    "push",
				Usage:   "Push local changes to remote",
				Aliases: []string{"p"},
				Action:  cmd.Push,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "no-confirm",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation",
					},
				},
			},
			{
				Name:    "sync",
				Usage:   "Sync to commit, downloading changes from remote",
				Aliases: []string{"to", "s"},
				Action:  cmd.Sync,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "no-confirm",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation",
					},
				},
			},
			{
				Name:    "reset",
				Usage:   "Reset all local changes",
				Aliases: []string{"r"},
				Action:  cmd.Reset,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "no-confirm",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation",
					},
				},
			},
			{
				Name:   "revert",
				Usage:  "Reset all local changes and sync to last commit",
				Action: cmd.Revert,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "no-confirm",
						Aliases: []string{"y"},
						Usage:   "Skip confirmation",
					},
				},
			},
			{
				Name:   "status",
				Usage:  "Print local project status",
				Action: cmd.PrintStatus,
			},
			{
				Name:    "branch",
				Aliases: []string{"b"},
				Subcommands: []*cli.Command{
					{
						Name:    "new",
						Aliases: []string{"n"},
						Usage:   "Create a new branch",
						Action:  cmd.NewBranch,
					},
					{
						Name:    "use",
						Aliases: []string{"u"},
						Usage:   "Switch to a different branch, syncing to its latest commit",
						Action:  cmd.UseBranch,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "no-confirm",
								Aliases: []string{"y"},
								Usage:   "Skip confirmation",
							},
						},
					},
					{
						Name:    "delete",
						Aliases: []string{"d"},
						Usage:   "Delete a branch",
						Action:  cmd.DeleteBranch,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "no-confirm",
								Aliases: []string{"y"},
								Usage:   "Skip confirmation",
							},
						},
					},
					{
						Name:    "set-default",
						Aliases: []string{"sd"},
						Usage:   "Set the default branch",
						Action:  cmd.SetDefaultBranch,
					},
				},
			},
			{
				Name:   "branches",
				Usage:  "List all branches in the project",
				Action: cmd.ListBranches,
			},
			{
				Name:   "history",
				Usage:  "List commit history",
				Action: cmd.PrintHistory,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "Limit number of commits",
						Value:   10,
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
