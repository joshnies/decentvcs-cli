package main

import (
	"log"
	"os"

	globalcmd "github.com/joshnies/decent/cmd/global"
	vcscmd "github.com/joshnies/decent/cmd/vcs"
	"github.com/joshnies/decent/config"
	"github.com/urfave/cli/v2"
)

func main() {
	// Initialize config
	config.InitConfig()

	// Initialize CLI app
	app := &cli.App{
		Name:      "decent",
		Usage:     "Decent CLI",
		Version:   "0.0.1",
		Copyright: "Copyright 2022 Joshua Nies. All rights reserved.",
		Authors: []*cli.Author{
			{
				Name:  "Joshua Nies",
				Email: "josh@decentvcs.com",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "login",
				Usage:  "Log in (required to use other commands)",
				Action: globalcmd.LogIn,
			},
			{
				Name:   "logout",
				Usage:  "Log out",
				Action: globalcmd.LogOut,
			},
			{
				Name:    "auth",
				Aliases: []string{"a"},
				Usage:   "Print current authentication state",
				Action:  globalcmd.PrintAuthState,
			},
			{
				Name:  "vcs",
				Usage: "DecentVCS commands",
				Subcommands: []*cli.Command{
					{
						Name:      "init",
						Usage:     "Initialize a new project",
						ArgsUsage: "[path]",
						Aliases:   []string{"i"},
						Action:    vcscmd.Init,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "name",
								Aliases: []string{"n"},
								Usage:   "Name of the project",
							},
						},
					},
					{
						Name:      "clone",
						Usage:     "Clone a project to your local machine",
						ArgsUsage: "[blob]",
						Action:    vcscmd.CloneProject,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "path",
								Value:   ".",
								Aliases: []string{"p"},
								Usage:   "Local path to directory for cloning project into",
							},
							&cli.StringFlag{
								Name:    "branch",
								Aliases: []string{"b"},
								Usage:   "Branch to clone",
							},
						},
					},
					{
						Name:    "changes",
						Usage:   "Print current changes",
						Aliases: []string{"c"},
						Action:  vcscmd.GetChanges,
					},
					{
						Name:      "push",
						Usage:     "Push local changes to remote",
						ArgsUsage: "[message?]",
						Aliases:   []string{"p"},
						Action: func(c *cli.Context) error {
							return vcscmd.Push(c)
						},
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "no-confirm",
								Aliases: []string{"y"},
								Usage:   "Skip confirmation",
							},
							&cli.StringFlag{
								Name:    "message",
								Aliases: []string{"m"},
								Usage:   "Commit message",
							},
						},
					},
					{
						Name:      "sync",
						Usage:     "Sync to commit, downloading changes from remote",
						ArgsUsage: "[commit_index?]",
						Aliases:   []string{"to", "s"},
						Action:    vcscmd.Sync,
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
						Action:  vcscmd.Reset,
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
						Action: vcscmd.Revert,
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
						Action: vcscmd.PrintStatus,
					},
					{
						Name:    "branch",
						Aliases: []string{"b"},
						Subcommands: []*cli.Command{
							{
								Name:      "new",
								Aliases:   []string{"n"},
								Usage:     "Create a new branch",
								ArgsUsage: "[name]",
								Action:    vcscmd.NewBranch,
							},
							{
								Name:      "use",
								Aliases:   []string{"u"},
								Usage:     "Switch to a different branch, syncing to its latest commit",
								ArgsUsage: "[name]",
								Action:    vcscmd.UseBranch,
								Flags: []cli.Flag{
									&cli.BoolFlag{
										Name:    "no-confirm",
										Aliases: []string{"y"},
										Usage:   "Skip confirmation",
									},
								},
							},
							{
								Name:      "delete",
								Aliases:   []string{"d"},
								Usage:     "Delete a branch",
								ArgsUsage: "[name]",
								Action:    vcscmd.DeleteBranch,
								Flags: []cli.Flag{
									&cli.BoolFlag{
										Name:    "no-confirm",
										Aliases: []string{"y"},
										Usage:   "Skip confirmation",
									},
								},
							},
							{
								Name:      "set-default",
								Aliases:   []string{"sd"},
								Usage:     "Set the default branch",
								ArgsUsage: "[name]",
								Action:    vcscmd.SetDefaultBranch,
							},
							{
								Name:      "rename",
								Usage:     "Rename a branch",
								ArgsUsage: "[old_name] [new_name]",
								Action:    vcscmd.RenameBranch,
							},
						},
					},
					{
						Name:   "branches",
						Usage:  "List all branches in the project",
						Action: vcscmd.ListBranches,
					},
					{
						Name:      "merge",
						Usage:     "Merge a branch into the current branch",
						ArgsUsage: "[branch]",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "no-confirm",
								Aliases: []string{"y"},
								Usage:   "Skip confirmation",
							},
							&cli.BoolFlag{
								Name:    "push",
								Aliases: []string{"p"},
								Usage:   "Push changes after merging",
							},
						},
						Action: vcscmd.Merge,
					},
					{
						Name:   "history",
						Usage:  "List commit history",
						Action: vcscmd.PrintHistory,
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
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
