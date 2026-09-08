package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

var (
	colorEnabled = detectColorEnabled(os.Stdout, os.Getenv)
)

func detectColorEnabled(w io.Writer, getenv func(string) string) bool {
	if v := getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return false
	}
	if getenv("TERM") == "dumb" {
		return false
	}
	if f, ok := w.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return (info.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func wrap(open, text string) string {
	if !colorEnabled || text == "" {
		return text
	}
	return open + text + reset
}

// Success prints a success message to stdout.
func Success(msg string) {
	fmt.Fprintln(os.Stdout, wrap(green, msg))
}

// Error prints an error message to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, wrap(red, "Error: "+msg))
}

// Warn prints a warning message to stderr.
func Warn(msg string) {
	fmt.Fprintln(os.Stderr, wrap(yellow, msg))
}

// Info prints an informational message to stdout.
func Info(msg string) {
	fmt.Fprintln(os.Stdout, wrap(cyan, "•")+" "+msg)
}

// Note prints a muted note to stdout.
func Note(msg string) {
	fmt.Fprintln(os.Stdout, wrap(dim, msg))
}

// PrintHarnessConnected prints the standard harness connected line.
func PrintHarnessConnected(label, model string) {
	display := displayModel(model)
	if display != "" {
		Success(fmt.Sprintf("%s → Fireworks · %s", label, display))
		return
	}
	Success(fmt.Sprintf("%s → Fireworks", label))
}

// PrintHarnessRestored prints the standard harness restored line.
func PrintHarnessRestored(label string) {
	fmt.Fprintf(os.Stdout, "%s restored to your previous setup.\n", label)
}

// PrintRestartHint prints a muted restart reminder.
func PrintRestartHint(msg string) {
	Note(msg)
}

func displayModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	model = strings.TrimSuffix(model, "[1m]")
	parts := strings.Split(model, "/")
	return parts[len(parts)-1]
}

// Bold returns bold text when color is enabled.
func Bold(text string) string {
	return wrap(bold, text)
}

// Accent returns cyan emphasized text when color is enabled.
func Accent(text string) string {
	return wrap(cyan, text)
}

// Muted returns dimmed text when color is enabled.
func Muted(text string) string {
	return wrap(dim, text)
}
