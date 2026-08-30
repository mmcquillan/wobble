# Execution Duration Specification

## Purpose

Controls how long a single `wobble` run lasts. A run has a **target runtime**
(fixed or sampled from a range) and an **absolute maximum duration** that the
process can never exceed, enforced by a watchdog independent of the workload.

## Requirements

### Requirement: Configurable target runtime

The utility SHALL accept a target runtime as either a single fixed Go duration or
an inclusive `min..max` range, and SHALL resolve the actual runtime for the run
exactly once, at startup, before any CPU-load work begins.

#### Scenario: Fixed target runtime

- **WHEN** the target runtime is configured as a single value `D`
- **THEN** the resolved runtime for the run is `D`
- **AND** the resolved value appears in the startup summary log record

#### Scenario: Ranged target runtime

- **WHEN** the target runtime is configured as `Dmin..Dmax` with `Dmin <= Dmax`
- **THEN** the resolved runtime is drawn from a uniform distribution over the
  closed interval `[Dmin, Dmax]`
- **AND** the draw is taken from the run's seeded RNG in the fixed draw order
  defined by the CLI capability, so it is reproducible for a given `(seed, config)`

#### Scenario: Degenerate range

- **WHEN** the target runtime is configured as `Dmin..Dmax` with `Dmin == Dmax`
- **THEN** the resolved runtime is that value

#### Scenario: Zero target runtime

- **WHEN** the resolved target runtime is `0`
- **THEN** no CPU-load workers are started
- **AND** the run still resolves and returns the success/failure exit code from
  the Exit Outcome capability

### Requirement: Absolute maximum duration

The utility SHALL accept an absolute maximum duration greater than zero, SHALL
treat it as a hard wall-clock ceiling measured from process start, and SHALL
begin an orderly shutdown no later than the moment that ceiling is reached.

#### Scenario: Target runtime below the maximum

- **WHEN** the resolved target runtime is less than the absolute maximum
- **THEN** the run proceeds for the resolved target runtime
- **AND** on reaching it the process stops CPU work and shuts down, exiting with
  the code from the Exit Outcome capability

#### Scenario: Target runtime exceeds the maximum

- **WHEN** the resolved or fixed target runtime is greater than the absolute
  maximum
- **THEN** the effective target runtime is clamped to the absolute maximum
- **AND** a warning log record notes the clamp and both values
- **AND** a clean shutdown at the ceiling still exits with the Exit Outcome code,
  not the watchdog code

#### Scenario: Default maximum when unset

- **WHEN** the user does not configure an absolute maximum
- **THEN** the CLI capability's documented default is applied
- **AND** the effective maximum appears in the startup summary log record

### Requirement: Watchdog termination

The utility SHALL run a watchdog that force-terminates the process if it is still
alive one grace period after the absolute maximum duration, so a stuck workload
or shutdown path can never cause an unbounded run.

#### Scenario: Shutdown completes within the grace period

- **WHEN** the process reaches the absolute maximum and finishes its shutdown
  within the configured grace period
- **THEN** the watchdog does not fire
- **AND** the exit code is the one selected by the Exit Outcome or CLI signal
  handling capability

#### Scenario: Shutdown overruns the grace period

- **WHEN** the process is still running one grace period after the absolute
  maximum duration
- **THEN** the watchdog force-exits the process with the `watchdog timeout` exit
  code defined in the CLI capability
- **AND** a log record records the terminal reason as `watchdog`

### Requirement: Elapsed time reporting

The utility SHALL record the actual elapsed wall-clock runtime and SHALL include
it in the final log record regardless of how the run terminated.

#### Scenario: Reported on every exit path

- **WHEN** the run ends by normal completion, by signal, or by watchdog
- **THEN** the final log record includes the elapsed runtime and the terminal
  reason (`completed`, `signal`, or `watchdog`)
