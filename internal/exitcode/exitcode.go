// Package exitcode holds the authoritative wobble exit-code table so that every
// other package refers to a code by name rather than by number. See
// openspec/specs/cli/spec.md, "Exit-code table".
package exitcode

const (
	// Success: run completed and the decided outcome is success.
	Success = 0
	// Usage: invalid configuration or CLI usage; no work performed.
	Usage = 2
	// Watchdog: process force-terminated one grace period after --max-duration.
	Watchdog = 124
	// SIGINT: terminated by SIGINT.
	SIGINT = 130
	// SIGTERM: terminated by SIGTERM.
	SIGTERM = 143
)

// Reserved reports whether c is a code this spec reserves for a specific
// termination reason, and therefore may not be used as --failure-code.
func Reserved(c int) bool {
	switch c {
	case Usage, Watchdog, SIGINT, SIGTERM:
		return true
	default:
		return false
	}
}
