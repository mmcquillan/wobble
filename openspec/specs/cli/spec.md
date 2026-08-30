# CLI Specification

## Purpose

Defines `wobble`'s configuration surface (flags and environment variables), input
validation, deterministic seeding and RNG draw order, signal handling, logging,
and the authoritative exit-code table that the other capabilities reference by
name.

## Requirements

### Requirement: Configuration inputs

Every tunable SHALL be settable by a command-line flag and MAY also be set by an
environment variable named `WOBBLE_<FLAG>` (uppercased, dashes to underscores).
When both are present the flag SHALL take precedence.

#### Scenario: Recognised options

- **WHEN** the utility parses its configuration
- **THEN** it accepts exactly the following options, with these defaults:

  | Flag | Env | Type | Default | Meaning |
  |------|-----|------|---------|---------|
  | `--duration` | `WOBBLE_DURATION` | Go duration, or `min..max` | `0s` | Target runtime; see Execution Duration |
  | `--max-duration` | `WOBBLE_MAX_DURATION` | Go duration `> 0` | `5m` | Absolute wall-clock ceiling |
  | `--grace` | `WOBBLE_GRACE` | Go duration `> 0` | `2s` | Shutdown grace period before the watchdog force-exits |
  | `--cpu` | `WOBBLE_CPU` | float `>= 0`, or `min..max` | `0` | CPU target in busy cores; see CPU Load |
  | `--workers` | `WOBBLE_WORKERS` | int `>= 0` | `0` (derive from `--cpu`) | Explicit load-worker count |
  | `--tolerance` | `WOBBLE_TOLERANCE` | float `> 0` | `0.1` | CPU accuracy tolerance in busy cores |
  | `--success-rate` | `WOBBLE_SUCCESS_RATE` | float in `[0,1]` | `1.0` | P(exit success); see Exit Outcome |
  | `--failure-code` | `WOBBLE_FAILURE_CODE` | int in `1..255`, not reserved | `1` | Exit code on a `failure` outcome |
  | `--seed` | `WOBBLE_SEED` | int64 | random | RNG seed; see Deterministic seeding |
  | `--log-format` | `WOBBLE_LOG_FORMAT` | `text` \| `json` | `text` | Structured log encoding on stderr |
  | `--verbose` / `-v` | `WOBBLE_VERBOSE` | bool | `false` | Extra logging, including the decided outcome at startup |
  | `--help` / `-h` | — | bool | — | Print usage and exit `0` |
  | `--version` | — | bool | — | Print version and exit `0` |

#### Scenario: Flag overrides environment

- **WHEN** `WOBBLE_CPU=1.0` is set and `--cpu 2.0` is passed
- **THEN** the resolved CPU target is based on `2.0`

#### Scenario: Help and version short-circuit

- **WHEN** `--help` or `--version` is passed
- **THEN** the utility prints the requested text to stdout and exits `0` without
  drawing any RNG values or starting any workers

### Requirement: Input validation

The utility SHALL validate all configuration before performing any work, and on
the first invalid value SHALL write a diagnostic to stderr and exit with the
`usage error` code, having started no workers and drawn no RNG values.

#### Scenario: Rejected configurations

- **WHEN** any of the following holds
  - a duration value is negative
  - a `min..max` range has `min > max`
  - `--max-duration` or `--grace` is less than or equal to zero
  - `--cpu` is negative
  - `--success-rate` is outside `[0, 1]`
  - `--failure-code` is outside `1..255` or equals a reserved code (`2`, `124`, `130`, `143`)
  - `--workers` or `--tolerance` is negative, or `--tolerance` is zero
  - `--log-format` is not `text` or `json`
  - any value fails to parse as its declared type
- **THEN** the utility exits with the `usage error` code and a message naming the
  offending option

#### Scenario: Resolved runtime may still exceed max after sampling

- **WHEN** a valid `--duration` range or fixed value resolves above a valid
  `--max-duration`
- **THEN** this is not a usage error; the Execution Duration capability clamps
  the effective runtime and logs a warning

### Requirement: Deterministic seeding and draw order

The utility SHALL create exactly one pseudo-random number generator per run, seed
it from the effective seed, and consume draws from it in this fixed order:

1. target runtime (only if `--duration` is a range)
2. CPU target (only if `--cpu` is a range)
3. success/failure outcome (always)

so that a given `(seed, full configuration)` pair fully determines the resolved
runtime, resolved CPU target, and decided outcome.

#### Scenario: Reproducing a run

- **WHEN** a run is executed with `--seed S` and configuration `X`
- **THEN** any later run with `--seed S` and configuration `X` resolves the same
  runtime, the same CPU target, and the same outcome

#### Scenario: Seed generated and logged when unset

- **WHEN** no `--seed` is provided
- **THEN** the utility generates a seed from a non-deterministic source
- **AND** the startup summary log record includes that seed so the run can be
  replayed with `--seed`

#### Scenario: Load-worker scheduling is not required to be deterministic

- **WHEN** two runs share a seed and configuration
- **THEN** their resolved runtime, CPU target, and outcome match
- **AND** exact per-worker busy/idle timing and observed CPU MAY differ

### Requirement: Startup summary log record

Before starting workers the utility SHALL emit exactly one structured log record
at info level containing at least: effective seed, resolved runtime, effective
max-duration, grace period, resolved CPU target, effective worker count, and
success rate. It SHALL include the decided outcome in this record only when
`--verbose` is set.

#### Scenario: Non-verbose startup

- **WHEN** `--verbose` is not set
- **THEN** the startup record contains the fields above and does not contain the
  decided outcome

### Requirement: Signal handling

On the first SIGINT or SIGTERM the utility SHALL stop all load workers, complete
shutdown within the grace period, and exit with `130` for SIGINT or `143` for
SIGTERM, overriding the decided success/failure exit code. On a second such
signal during shutdown it SHALL exit immediately with the same code.

#### Scenario: Graceful stop on SIGTERM

- **WHEN** SIGTERM is received mid-run
- **THEN** workers stop, a final log record is emitted with terminal reason
  `signal`, and the process exits `143`

#### Scenario: Impatient operator

- **WHEN** a second SIGINT arrives before shutdown finishes
- **THEN** the process exits `130` without waiting for the remaining shutdown
  work

#### Scenario: Signal after the outcome would have been success

- **WHEN** the decided outcome is `success` and SIGINT arrives before the
  resolved runtime elapses
- **THEN** the process exits `130`, not `0`

### Requirement: Final log record

On every exit path except `--help`, `--version`, and usage errors, the utility
SHALL emit exactly one structured log record containing at least: elapsed
runtime, resolved CPU target, observed average CPU (when any worker ran), decided
outcome, terminal reason (`completed` \| `signal` \| `watchdog`), and the exit
code.

#### Scenario: Completed run

- **WHEN** a run reaches its resolved runtime
- **THEN** the final record's terminal reason is `completed` and its exit code
  matches the Exit Outcome mapping

#### Scenario: Watchdog run

- **WHEN** the watchdog force-terminates the process
- **THEN** a final record with terminal reason `watchdog` and exit code `124` is
  emitted before the process exits

### Requirement: Exit-code table

The utility SHALL use exactly these process exit codes and no others:

| Code | Name | Condition |
|------|------|-----------|
| `0` | success | Run completed; decided outcome is `success` |
| `--failure-code` (default `1`) | failure | Run completed; decided outcome is `failure` |
| `2` | usage error | Invalid configuration or CLI usage; no work performed |
| `124` | watchdog timeout | Process force-terminated one grace period after `--max-duration` |
| `130` | sigint | Terminated by SIGINT |
| `143` | sigterm | Terminated by SIGTERM |

#### Scenario: Codes are stable identifiers

- **WHEN** another capability's spec refers to the `usage error`, `watchdog
  timeout`, `sigint`, or `sigterm` code
- **THEN** it means the corresponding row of this table

#### Scenario: Failure code cannot shadow a reserved code

- **WHEN** `--failure-code` is set to any of `2`, `124`, `130`, `143`
- **THEN** the utility exits `2` with a diagnostic (per Input validation)
