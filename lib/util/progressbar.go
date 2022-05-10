package util

import (
	"github.com/schollz/progressbar/v3"
)

// Create new progress bar with custom options.
func NewProgressBar(max int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(
		max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
	)
}
