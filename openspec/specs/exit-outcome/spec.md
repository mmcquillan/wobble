# Exit Outcome Specification

## Purpose

Decides, once per run, whether `wobble` reports **success** or **failure**, based
on a configurable success probability, and maps that decision onto the process
exit code. This is the "percent of success versus failure" behaviour.

## Requirements

### Requirement: Success rate draw

The utility SHALL accept a success rate `p` in the closed interval `[0, 1]` and
SHALL decide the run outcome with a single Bernoulli trial against the run's
seeded RNG: `success` with probability `p`, `failure` with probability `1 - p`.

#### Scenario: Typical rate

- **WHEN** `p = 0.9` and many runs are executed, each with a distinct seed
- **THEN** approximately 90% of runs decide `success` and 10% decide `failure`
  within normal sampling variance

#### Scenario: Always succeed

- **WHEN** `p = 1`
- **THEN** every run decides `success`

#### Scenario: Always fail

- **WHEN** `p = 0`
- **THEN** every run decides `failure`

#### Scenario: Default rate

- **WHEN** the user does not configure a success rate
- **THEN** the CLI capability's documented default is used

### Requirement: Decision timing

The utility SHALL make the outcome draw at startup, in the fixed RNG draw order
defined by the CLI capability (after the runtime draw and the CPU-target draw),
and SHALL hold the decision unchanged for the rest of the run.

#### Scenario: Reproducible outcome

- **WHEN** two runs use the same seed and the same configuration
- **THEN** they decide the same outcome

#### Scenario: Outcome not revealed early by default

- **WHEN** verbose logging is not enabled
- **THEN** the startup summary log record does not state the decided outcome
- **AND** the outcome first appears in the final log record at exit

#### Scenario: Outcome revealed under verbose

- **WHEN** verbose logging is enabled
- **THEN** the decided outcome is included in the startup summary log record

### Requirement: Outcome-to-exit-code mapping

On a run that completes normally, the utility SHALL exit `0` when the decided
outcome is `success` and SHALL exit with the configured failure exit code when
the decided outcome is `failure`.

#### Scenario: Successful completion

- **WHEN** the outcome is `success` and the run reaches its resolved runtime
- **THEN** the process exits with code `0`
- **AND** the final log record states `outcome=success` and the terminal reason
  `completed`

#### Scenario: Failed completion

- **WHEN** the outcome is `failure` and the run reaches its resolved runtime
- **THEN** the process exits with the configured failure exit code (default `1`)
- **AND** the final log record states `outcome=failure` and the terminal reason
  `completed`

### Requirement: Configurable failure exit code

The utility SHALL accept a failure exit code in the range `1..255`, SHALL default
it to `1`, and SHALL reject any value that collides with a code reserved by the
CLI capability's exit-code table (`2`, `124`, `130`, `143`) as a configuration
error.

#### Scenario: Custom failure code

- **WHEN** the failure exit code is set to `7` and the outcome is `failure`
- **THEN** the process exits with code `7`

#### Scenario: Reserved failure code rejected

- **WHEN** the failure exit code is set to `2`, `124`, `130`, or `143`
- **THEN** the utility exits with the `usage error` code and a diagnostic, having
  performed no work

### Requirement: Outcome overridden by abnormal termination

When a run is terminated by signal or by the watchdog, the utility SHALL exit
with the code for that termination reason and SHALL NOT substitute the decided
success/failure exit code.

#### Scenario: Signal during a run that had decided success

- **WHEN** the outcome is `success` but SIGTERM is received before the resolved
  runtime elapses
- **THEN** the process exits with the SIGTERM code from the CLI capability, not
  `0`
- **AND** the final log record states `outcome=success` and terminal reason
  `signal`
