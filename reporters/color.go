// Package reporters: color.go emits bright 16-color SGR codes for the severity
// column on TTY output. Stdlib only — no dependencies. We never emit truecolor
// or 256-color codes, so no downsampling pipeline is needed: when color is
// disabled we simply don't write the codes.
package reporters

import (
	"io"
	"os"
	"strings"

	"github.com/open-rpc/openrpc-linter/types"
)

// Bright ANSI 16 foreground codes. Bright variants render more reliably than
// 31/33/34 across themes (33 in particular often shows as brown).
const (
	ansiReset        = "\x1b[0m"
	ansiBrightRed    = "\x1b[91m"
	ansiBrightYellow = "\x1b[93m"
	ansiBrightBlue   = "\x1b[94m"
	// 90 = "bright black" — most terminals render it as dark grey.
	ansiDarkGrey = "\x1b[90m"
	// 37 = "white" — on dark themes this is the light-grey foreground.
	ansiLightGrey = "\x1b[37m"
)

// severityColWidth is the visible width of the severity column. It must be at
// least len("warning") = 7 so all labels line up after padding.
const severityColWidth = 7

func severityColorCode(severity types.Severity) string {
	switch effectiveSeverity(severity) {
	case types.SeverityWarn:
		return ansiBrightYellow
	case types.SeverityInfo:
		return ansiBrightBlue
	default:
		return ansiBrightRed
	}
}

// supportsColor decides whether to emit ANSI codes for w. It follows the
// no-color.org and bixense CLICOLOR conventions, plus a TTY check. Order
// matters: explicit opt-out wins, then explicit opt-in, then heuristics.
func supportsColor(w io.Writer) bool {
	// Per https://no-color.org/: presence and non-empty value disables color.
	// Empty string is treated as "not set" so tests can clear it with t.Setenv.
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	if v := os.Getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	if v := os.Getenv("CLICOLOR_FORCE"); v != "" && v != "0" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// formatSeverityCol returns a fixed-width severity cell. When colorEnabled is
// true the label is wrapped in ANSI codes but the trailing pad stays uncolored
// so column alignment is preserved.
func formatSeverityCol(severity types.Severity, colorEnabled bool) string {
	return styledCell(severityLabel(severity), severityColWidth, severityColorCode(severity), colorEnabled)
}

// styledCell renders s left-padded to width and, when enabled is true,
// wraps the visible text (not the trailing pad) with the given ANSI code.
// Passing an empty code or enabled=false yields the plain padded cell so
// column alignment is preserved regardless of color state.
func styledCell(s string, width int, code string, enabled bool) string {
	pad := width - len(s)
	if pad < 0 {
		pad = 0
	}
	if !enabled || code == "" {
		return s + strings.Repeat(" ", pad)
	}
	return code + s + ansiReset + strings.Repeat(" ", pad)
}

// colorize wraps s with code (and a reset) when enabled is true. Used for
// the trailing rule-id cell where no padding is required.
func colorize(s, code string, enabled bool) string {
	if !enabled || code == "" {
		return s
	}
	return code + s + ansiReset
}
