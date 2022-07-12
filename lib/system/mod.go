package system

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Open the default browser with the given URL.
func OpenBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform encountered while attempting to open browser")
	}
	if err != nil {
		log.Fatal(err)
	}
}

// Get temp directory specific to Decent.
// The directory is created if it doesn't exist.
func GetTempDir() string {
	// Get user home dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Get temp dir
	tempDir := filepath.Join(homeDir, "decent", "tmp")
	if err != nil {
		log.Fatal(err)
	}

	// Create temp dir if it doesn't exist
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		err = os.MkdirAll(tempDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	return tempDir
}

// Returns a slice of all files in a directory recursively.
func ListFiles(dir string) ([]string, error) {
	var res []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		res = append(res, path)
		return nil
	})
	return res, err
}

// Searches for the specified file within any directory above the current working directory.
// Returns the path to the file if found, otherwise an error.
func FindFileUpwards(filename string) (string, error) {
	// Look for project config file
	//
	// Get absolute current directory as initial search path
	searchPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// While config foundPath is empty...
	var foundPath string
	for foundPath == "" {
		// Check if file exists
		searchPathWithFile := filepath.Join(searchPath, filename)
		if _, err := os.Stat(searchPathWithFile); err != nil {
			// If end of search path, return error
			if strings.Split(searchPath, string(os.PathSeparator))[0] == searchPath {
				return "", errors.New("not found")
			}

			// Not found yet (or an error occurred), move up one directory
			searchPath = filepath.Dir(searchPath)
		} else {
			// File was found, break
			foundPath = searchPathWithFile
			break
		}
	}

	return foundPath, nil
}
