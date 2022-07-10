package vcs

import (
	"bufio"
	"os"
	"strings"

	"github.com/joshnies/decent/constants"
	"github.com/joshnies/decent/lib/system"
)

// Returns file patterns from the closest DecentVCS ignore file (using an upwards file search).
//
// If an ignore file isn't found, returns nil.
func GetIgnoredFilePatterns() ([]string, error) {
	// Find ignore file
	ignoreFilePath, err := system.FindFileUpwards(constants.IgnoreFileName)
	if err != nil {
		return nil, err
	}

	// Read file
	ignoreFile, err := os.Open(ignoreFilePath)
	if err != nil {
		return nil, err
	}
	defer ignoreFile.Close()

	ignoredFilePatterns := []string{}
	scanner := bufio.NewScanner(ignoreFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ignoredFilePatterns = append(ignoredFilePatterns, line)
		}
	}

	return ignoredFilePatterns, nil
}
