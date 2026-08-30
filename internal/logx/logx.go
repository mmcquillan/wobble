// Package logx builds the structured logger wobble writes to stderr. Format is
// "text" or "json"; --verbose lowers the level so debug records (and the decided
// outcome in the startup record) are emitted.
package logx

import (
	"io"
	"log/slog"
)

// New returns a logger writing to w. format must be "text" or "json"; any other
// value falls back to text (config validation rejects it before this is called).
func New(w io.Writer, format string, verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	if format == "json" {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}
