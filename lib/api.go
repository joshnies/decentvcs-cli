package lib

import (
	"fmt"

	"github.com/joshnies/qc-cli/config"
)

// Build API URL for given path.
func BuildURL(path string) string {
	return config.I.API.Host + "/" + path
}

// Build API URL for given path.
func BuildURLf(path string, v ...any) string {
	return config.I.API.Host + "/" + fmt.Sprintf(path, v...)
}
