package console

import (
	"fmt"
	"os"

	"github.com/TwiN/go-color"
	"github.com/joshnies/decent/constants"
)

type LogLevel int64

const (
	LogLevelVerbose LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

// Log verbose message to console.
// Verbose mode must be enabled for message to be printed.
func Verbose(message string, vars ...any) {
	// Get env var directly to prevent circular import
	if os.Getenv(constants.VerboseEnvVar) != "1" {
		return
	}

	fmt.Printf(color.Ize(color.Gray, message+"\n"), vars...)
}

// Log success message to console.
func Success(message string, vars ...any) {
	fmt.Printf(color.Ize(color.Green, message+"\n"), vars...)
}

// Log info message to console.
func Info(message string, vars ...any) {
	fmt.Printf(color.Ize(color.Cyan, message+"\n"), vars...)
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

// Log error message to console, but only if verbose mode is enabled.
func ErrorPrintV(message string, vars ...any) {
	// Get env var directly to prevent circular import
	if os.Getenv(constants.VerboseEnvVar) != "1" {
		return
	}

	ErrorPrint(message, vars...)
}

// Log fatal error message to console, exiting the process.
func Fatal(message string, vars ...any) {
	ErrorPrint(message, vars...)
	os.Exit(1)
}
