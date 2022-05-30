package system

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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

// Get quanta-specific temp directory.
// The directory is created if it doesn't exist.
func GetTempDir() string {
	// Get user home dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Get temp dir
	tempDir := filepath.Join(homeDir, "quanta", "tmp")
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
