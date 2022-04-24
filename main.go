package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/joshnies/qc-cli/cmd"
	"github.com/joshnies/qc-cli/config"
	"github.com/urfave/cli/v2"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize config
	config.InitConfig()

	// Initialize CLI app
	app := &cli.App{
		Name:    "qc",
		Usage:   "Quanta Control CLI",
		Version: "0.0.1",
		Commands: []*cli.Command{
			{
				Name:    "init",
				Usage:   "Initialize a new project",
				Aliases: []string{"i"},
				Action:  cmd.Init,
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
