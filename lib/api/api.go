package api

import "github.com/joshnies/qc-cli/config"

// Build API URL for given path.
func BuildURL(path string) string {
	return config.I.API.Host + "/" + path
}
