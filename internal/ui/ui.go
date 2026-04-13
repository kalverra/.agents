// Package ui provides a centralized way to handle user output.
package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	// AIOutput determines if the output should be formatted for LLMs.
	AIOutput bool
)

// SetAIOutput sets the global AIOutput flag.
func SetAIOutput(b bool) {
	AIOutput = b
}

// Printf prints a formatted string to stdout if not in AI output mode.
// If in AI output mode, it can be silenced or formatted differently.
func Printf(format string, a ...any) {
	if !AIOutput {
		fmt.Printf(format, a...)
	}
}

// Println prints a string to stdout if not in AI output mode.
func Println(a ...any) {
	if !AIOutput {
		fmt.Println(a...)
	}
}

// SuccessPrintf prints a success message.
func SuccessPrintf(format string, a ...any) {
	if AIOutput {
		fmt.Printf("SUCCESS: "+format, a...)
	} else {
		fmt.Printf("✅ "+format, a...)
	}
}

// ErrorPrintf prints an error message to stderr.
func ErrorPrintf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format, a...)
}

// WarnPrintf prints a warning message to stderr.
func WarnPrintf(format string, a ...any) {
	if !AIOutput {
		fmt.Fprintf(os.Stderr, "⚠️  "+format, a...)
	} else {
		fmt.Fprintf(os.Stderr, "WARNING: "+format, a...)
	}
}

// VerbosePrintf prints only if not in AI mode.
func VerbosePrintf(verbose bool, format string, a ...any) {
	if verbose && !AIOutput {
		fmt.Printf(format, a...)
	}
}

// PrintJSON prints a JSON object to stdout.
func PrintJSON(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}
