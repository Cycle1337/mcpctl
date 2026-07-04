package main

import (
	"fmt"
	"os"
)

// ANSI styles. Kept to a minimum so output stays readable on every theme.
const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[90m"
	cGreen  = "\033[32m"
	cCyan   = "\033[36m"
	cYellow = "\033[33m"
	cRed    = "\033[31m"
)

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// paint wraps s in style only when color is enabled, so piped/CI output
// stays clean (no escape codes). Set MCPCTL_COLOR=always to force on,
// MCPCTL_COLOR=never to force off; default is auto (tty detection).
func paint(s, style string) string {
	switch os.Getenv("MCPCTL_COLOR") {
	case "always":
	case "never":
		return s
	default:
		if !isTTY(os.Stdout) {
			return s
		}
	}
	return style + s + cReset
}

func faint(s string) string  { return paint(s, cDim) }
func bold(s string) string   { return paint(s, cBold) }
func green(s string) string  { return paint(s, cGreen) }
func cyan(s string) string   { return paint(s, cCyan) }
func yellow(s string) string { return paint(s, cYellow) }
func red(s string) string    { return paint(s, cRed) }

// prompt renders a shell prompt line for demo output.
func prompt() string {
	return faint("$ ")
}

// keep fmt alive for future formatted color helpers
var _ = fmt.Sprintf
