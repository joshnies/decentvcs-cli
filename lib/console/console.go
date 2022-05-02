package console

import (
	"fmt"

	"github.com/TwiN/go-color"
)

type LogLevel int64

const (
	LogLevelVerbose LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

type LogOptions struct {
	Message string
	Vars    []interface{}
}

// Log verbose message to console.
// `VERBOSE` environment variable must be set to `1` for message to be printed.
func Verbose(message string, vars ...any) {
	fmt.Printf(color.Ize(color.Gray, "[V] "+message+"\n"), vars...)
}

// Log info message to console.
func Info(message string, vars ...any) {
	fmt.Printf(color.Ize(color.Cyan, "[INFO] "+message+"\n"), vars...)
}

// Log warning message to console.
func Warning(message string, vars ...any) {
	fmt.Printf(color.Ize(color.Yellow, "[WARNING] "+message+"\n"), vars...)
}

// Log error message to console.
func Error(message string, vars ...any) error {
	return fmt.Errorf(color.Ize(color.Red, "[ERROR] "+message+"\n"), vars...)
}

// Log error message to console.
func ErrorPrint(message string, vars ...any) {
	fmt.Printf(color.Ize(color.Red, "[ERROR] "+message+"\n"), vars...)
}
