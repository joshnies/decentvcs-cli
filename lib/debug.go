package lib

import (
	"fmt"

	"github.com/TwiN/go-color"
	"github.com/joshnies/qc-cli/config"
)

type LogLevel int64

const (
	Info LogLevel = iota
	Warning
	Error
)

type LogOptions struct {
	Level       LogLevel
	Str         string
	Vars        []interface{}
	VerboseStr  string
	VerboseVars []interface{}
}

// Custom logging function with support for verbosity levels.
func Log(o LogOptions) error {
	if config.I.Verbose && o.VerboseStr != "" {
		// Print verbose message in addition to user-facing message
		fmt.Printf("[VERBOSE] "+o.VerboseStr+"\n", o.VerboseVars...)
	}

	switch o.Level {
	case Info:
		fmt.Printf(color.Ize(color.Cyan, "[INFO] "+o.Str+"\n"), o.Vars...)
	case Warning:
		fmt.Printf(color.Ize(color.Yellow, "[WARNING] "+o.Str+"\n"), o.Vars...)
	case Error:
		return fmt.Errorf(color.Ize(color.Red, "[ERROR] "+o.Str+"\n"), o.Vars...)
	}

	return nil
}
