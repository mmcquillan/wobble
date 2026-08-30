package outcome_test

import (
	"testing"

	"github.com/wobble/internal/outcome"
	"github.com/wobble/internal/rng"
)

func TestDecideAndExitCode(t *testing.T) {
	if outcome.Decide(true) != outcome.Success || outcome.Decide(false) != outcome.Failure {
		t.Fatal("Decide mapping wrong")
	}
	if outcome.Success.String() != "success" || outcome.Failure.String() != "failure" {
		t.Fatal("String mapping wrong")
	}
	if got := outcome.ExitCode(outcome.Success, 7); got != 0 {
		t.Errorf("ExitCode(success) = %d, want 0", got)
	}
	if got := outcome.ExitCode(outcome.Failure, 7); got != 7 {
		t.Errorf("ExitCode(failure, 7) = %d, want 7", got)
	}
}

func TestFrequencyMatchesSuccessRate(t *testing.T) {
	const (
		n = 20000
		p = 0.9
	)
	successes := 0
	for i := 0; i < n; i++ {
		// A distinct seed per run, as in the spec scenario.
		if outcome.Decide(rng.New(uint64(i)*2654435761+1).DrawOutcome(p)).String() == "success" {
			successes++
		}
	}
	rate := float64(successes) / n
	if rate < 0.86 || rate < p-0.04 || rate > 0.94 {
		t.Errorf("observed success rate %.4f, want ~%.2f", rate, p)
	}
}
