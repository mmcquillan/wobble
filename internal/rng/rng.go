// Package rng provides the single seeded pseudo-random source used for a wobble
// run, plus the three draws taken from it in a fixed order:
//
//  1. target runtime  (only when --duration is a range)
//  2. CPU target      (only when --cpu is a range)
//  3. success/failure outcome (always)
//
// See openspec/specs/cli/spec.md, "Deterministic seeding and draw order".
package rng

import (
	"math/rand/v2"
	"time"
)

// Source is one run's random generator. A given (seed, config) pair fully
// determines every draw taken from it.
type Source struct {
	r *rand.Rand
}

// New returns a Source seeded deterministically from seed.
func New(seed uint64) *Source {
	// The second PCG parameter is a fixed stream selector; only seed varies.
	return &Source{r: rand.New(rand.NewPCG(seed, 0x9E3779B97F4A7C15))}
}

// DrawRuntime resolves the target runtime. When isRange is false it returns min
// and consumes no randomness; otherwise it returns a uniform draw over the
// closed interval [min, max].
func (s *Source) DrawRuntime(min, max time.Duration, isRange bool) time.Duration {
	if !isRange {
		return min
	}
	span := float64(max - min)
	return min + time.Duration(s.r.Float64()*span)
}

// DrawCPU resolves the CPU target in busy cores. When isRange is false it
// returns min and consumes no randomness; otherwise it returns a uniform draw
// over [min, max].
func (s *Source) DrawCPU(min, max float64, isRange bool) float64 {
	if !isRange {
		return min
	}
	return min + s.r.Float64()*(max-min)
}

// DrawOutcome always consumes exactly one draw and reports whether the run
// should report success. p == 1 always succeeds; p == 0 always fails.
func (s *Source) DrawOutcome(p float64) bool {
	return s.r.Float64() < p
}
