package console

import (
	"fmt"
	"os"

	"github.com/TwiN/go-color"
)

type LogLevel int64

const (
	LogLevelVerbose LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

// Log verbose message to console.
// `VERBOSE` environment variable must be set to `1` for message to be printed.
func Verbose(message string, vars ...any) {
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

// Log fatal error message to console, exiting the process.
func Fatal(message string, vars ...any) {
	ErrorPrint(message, vars...)
	os.Exit(1)
}
