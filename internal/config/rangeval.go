package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const rangeSep = ".."

// DurationSpec is a resolved-at-startup target runtime: a fixed value, or an
// inclusive [Min, Max] range to sample from.
type DurationSpec struct {
	Min, Max time.Duration
	IsRange  bool
}

// CPUSpec is a CPU target in busy cores: a fixed value, or an inclusive
// [Min, Max] range to sample from.
type CPUSpec struct {
	Min, Max float64
	IsRange  bool
}

// parseDurationSpec parses "10s" or "5s..30s". A degenerate range (Min == Max)
// collapses to a fixed value so no RNG draw is consumed for it.
func parseDurationSpec(s string) (DurationSpec, error) {
	s = strings.TrimSpace(s)
	if lo, hi, ok := strings.Cut(s, rangeSep); ok {
		min, err := time.ParseDuration(strings.TrimSpace(lo))
		if err != nil {
			return DurationSpec{}, fmt.Errorf("invalid lower bound %q: %w", strings.TrimSpace(lo), err)
		}
		max, err := time.ParseDuration(strings.TrimSpace(hi))
		if err != nil {
			return DurationSpec{}, fmt.Errorf("invalid upper bound %q: %w", strings.TrimSpace(hi), err)
		}
		if min < 0 || max < 0 {
			return DurationSpec{}, fmt.Errorf("must not be negative")
		}
		if min > max {
			return DurationSpec{}, fmt.Errorf("lower bound %s exceeds upper bound %s", min, max)
		}
		return DurationSpec{Min: min, Max: max, IsRange: min != max}, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return DurationSpec{}, err
	}
	if d < 0 {
		return DurationSpec{}, fmt.Errorf("must not be negative")
	}
	return DurationSpec{Min: d, Max: d}, nil
}

// parseCPUSpec parses "1.5" or "0.5..3". A degenerate range collapses to a
// fixed value.
func parseCPUSpec(s string) (CPUSpec, error) {
	s = strings.TrimSpace(s)
	if lo, hi, ok := strings.Cut(s, rangeSep); ok {
		min, err := strconv.ParseFloat(strings.TrimSpace(lo), 64)
		if err != nil {
			return CPUSpec{}, fmt.Errorf("invalid lower bound %q: %w", strings.TrimSpace(lo), err)
		}
		max, err := strconv.ParseFloat(strings.TrimSpace(hi), 64)
		if err != nil {
			return CPUSpec{}, fmt.Errorf("invalid upper bound %q: %w", strings.TrimSpace(hi), err)
		}
		if min < 0 || max < 0 {
			return CPUSpec{}, fmt.Errorf("must be >= 0")
		}
		if min > max {
			return CPUSpec{}, fmt.Errorf("lower bound %g exceeds upper bound %g", min, max)
		}
		return CPUSpec{Min: min, Max: max, IsRange: min != max}, nil
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return CPUSpec{}, err
	}
	if v < 0 {
		return CPUSpec{}, fmt.Errorf("must be >= 0")
	}
	return CPUSpec{Min: v, Max: v}, nil
}
