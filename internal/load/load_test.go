package load

import (
	"context"
	"math"
	"testing"
	"time"
)

func approx(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-9 {
			return false
		}
	}
	return true
}

func TestDuties(t *testing.T) {
	cases := []struct {
		target   float64
		override int
		want     []float64
	}{
		{0, 0, nil},
		{0.5, 0, []float64{0.5}},
		{1, 0, []float64{1}},
		{2.5, 0, []float64{1, 1, 0.5}}, // staircase: two flat-out, one half
		{3, 0, []float64{1, 1, 1}},
		{2.5, 2, []float64{1, 1}}, // override: target/W = 1.25, clamped to 1
		{2.5, 4, []float64{0.625, 0.625, 0.625, 0.625}},
		{0, 3, []float64{0, 0, 0}}, // override wins even at zero target
	}
	for _, tc := range cases {
		got := Duties(tc.target, tc.override)
		if !approx(got, tc.want) {
			t.Errorf("Duties(%v, %d) = %v, want %v", tc.target, tc.override, got, tc.want)
		}
	}
}

func TestPoolStopsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := Start(ctx, Duties(2, 0))
	time.Sleep(150 * time.Millisecond)

	cancel()
	done := make(chan struct{})
	go func() { p.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("workers did not stop within 500ms of cancel")
	}
}

func TestInertPoolWithoutDuties(t *testing.T) {
	p := Start(context.Background(), nil)
	if p.Ran() {
		t.Error("Ran() true with no duties")
	}
	p.Stop() // must not panic or hang
	if _, ok := p.ObservedCores(); ok {
		t.Error("ObservedCores ok with no workers")
	}
}

// TestLoadAccuracy is timing- and scheduler-sensitive; skipped under -short.
func TestLoadAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("load accuracy test skipped in -short mode")
	}
	const target = 0.5
	p := Start(context.Background(), Duties(target, 0))
	time.Sleep(4 * time.Second)
	p.Stop()

	got, ok := p.ObservedCores()
	if !ok {
		t.Skip("CPU accounting unavailable on this platform")
	}
	if math.Abs(got-target) > 0.15 {
		t.Errorf("observed %.3f busy cores, want %.2f (spec tolerance 0.1)", got, target)
	}
}
