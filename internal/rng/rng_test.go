package rng

import (
	"testing"
	"time"
)

func TestReproducible(t *testing.T) {
	draw := func() (time.Duration, float64, bool) {
		s := New(12345)
		return s.DrawRuntime(time.Second, 5*time.Second, true),
			s.DrawCPU(0.5, 2.5, true),
			s.DrawOutcome(0.9)
	}
	r1, c1, o1 := draw()
	r2, c2, o2 := draw()
	if r1 != r2 || c1 != c2 || o1 != o2 {
		t.Errorf("same seed produced different draws: (%v,%v,%v) vs (%v,%v,%v)", r1, c1, o1, r2, c2, o2)
	}
	if r1 < time.Second || r1 > 5*time.Second {
		t.Errorf("runtime %v outside [1s,5s]", r1)
	}
	if c1 < 0.5 || c1 > 2.5 {
		t.Errorf("cpu %v outside [0.5,2.5]", c1)
	}
}

func TestScalarConsumesNoRandomness(t *testing.T) {
	// A fixed (non-range) runtime and CPU must not advance the generator, so the
	// outcome draw is identical to a generator that only draws the outcome.
	withScalars := New(999)
	withScalars.DrawRuntime(3*time.Second, 3*time.Second, false)
	withScalars.DrawCPU(1.0, 1.0, false)
	a := withScalars.DrawOutcome(0.5)

	outcomeOnly := New(999)
	b := outcomeOnly.DrawOutcome(0.5)

	if a != b {
		t.Errorf("scalar draws consumed randomness: %v vs %v", a, b)
	}
}

func TestRangeConsumesRandomness(t *testing.T) {
	// A ranged runtime draw must advance the generator, shifting later draws.
	withRange := New(999)
	withRange.DrawRuntime(0, 10*time.Second, true)
	seq := make([]bool, 32)
	for i := range seq {
		seq[i] = withRange.DrawOutcome(0.5)
	}

	outcomeOnly := New(999)
	same := true
	for i := range seq {
		if seq[i] != outcomeOnly.DrawOutcome(0.5) {
			same = false
			break
		}
	}
	if same {
		t.Error("ranged runtime draw did not advance the generator")
	}
}

func TestDrawOutcomeExtremes(t *testing.T) {
	s := New(1)
	for i := 0; i < 5000; i++ {
		if !s.DrawOutcome(1.0) {
			t.Fatal("p=1 produced a failure")
		}
	}
	s = New(1)
	for i := 0; i < 5000; i++ {
		if s.DrawOutcome(0.0) {
			t.Fatal("p=0 produced a success")
		}
	}
}
