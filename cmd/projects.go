package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func Init(c *cli.Context) error {
	// Get absolute file path
	path := c.Args().First()
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Create directories if they don't exist
	err = os.MkdirAll(absPath, os.ModePerm)
	if err != nil {
		log.Fatalf("\"%s\" is an existing file, aborting...", absPath)
	}

	// TODO: Create project in API
	// TODO: Create QC project file
	// TODO: Create QC history file
	// TODO: Create QC ignore file

	return nil
}
