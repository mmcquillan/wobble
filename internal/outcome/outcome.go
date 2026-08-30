// Package outcome models the per-run success/failure decision and its mapping
// onto a process exit code. See openspec/specs/exit-outcome/spec.md.
package outcome

// Outcome is the decided result of a run.
type Outcome int

const (
	// Success maps to exit code 0.
	Success Outcome = iota
	// Failure maps to the configured --failure-code.
	Failure
)

func (o Outcome) String() string {
	if o == Success {
		return "success"
	}
	return "failure"
}

// Decide converts a Bernoulli trial result into an Outcome.
func Decide(success bool) Outcome {
	if success {
		return Success
	}
	return Failure
}

// ExitCode returns the process exit code for a run that completed normally:
// 0 for success, failureCode for failure. It is not used on the signal or
// watchdog paths, which override the outcome.
func ExitCode(o Outcome, failureCode int) int {
	if o == Success {
		return 0
	}
	return failureCode
}
