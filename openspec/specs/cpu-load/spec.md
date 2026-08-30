# CPU Load Specification

## Purpose

Makes a `wobble` run consume a configurable, optionally randomized amount of CPU,
held roughly constant from startup until the run ends. Load is expressed in units
of **busy cores**: `1.0` means one CPU fully saturated, `0.25` a quarter of one
core, `3.0` three cores' worth spread across workers.

## Requirements

### Requirement: Configurable CPU target

The utility SHALL accept a CPU target in busy cores as either a single
non-negative number or an inclusive `min..max` range, and SHALL resolve the
actual target for the run exactly once, at startup, using the run's seeded RNG in
the fixed draw order defined by the CLI capability.

#### Scenario: Fixed CPU target

- **WHEN** the CPU target is configured as a single value `C`
- **THEN** the run aims to keep average utilisation near `C` busy cores
- **AND** the resolved target appears in the startup summary log record

#### Scenario: Ranged CPU target

- **WHEN** the CPU target is configured as `Cmin..Cmax` with `0 <= Cmin <= Cmax`
- **THEN** the resolved target is drawn uniformly from `[Cmin, Cmax]`
- **AND** the draw is reproducible for a given `(seed, config)`

#### Scenario: Zero CPU target

- **WHEN** the resolved CPU target is `0`
- **THEN** no load-generating workers busy-loop
- **AND** the run still lasts its resolved runtime and returns the Exit Outcome
  code

#### Scenario: Target exceeds available parallelism

- **WHEN** the resolved CPU target in cores is greater than the number of usable
  CPUs (`GOMAXPROCS`)
- **THEN** the effective target is clamped to the usable CPU count
- **AND** a warning log record notes the clamp and both values

### Requirement: Load generation model

The utility SHALL generate load with a pool of worker goroutines, each pinned to
busy-looping for a fraction of a short, fixed control interval and idling for the
remainder, such that the summed duty cycle across workers approximates the
resolved CPU target.

#### Scenario: Worker count derived from the target

- **WHEN** the user does not override the worker count
- **THEN** the utility starts `ceil(target)` workers (at least one when the
  target is greater than zero)
- **AND** the combined duty cycle of those workers targets the resolved value

#### Scenario: Fractional-core target

- **WHEN** the resolved target is `0.5` cores
- **THEN** a single worker busy-loops for approximately half of each control
  interval and idles for the rest

#### Scenario: Multi-core target

- **WHEN** the resolved target is `2.5` cores on a host with at least 3 usable
  CPUs
- **THEN** two workers busy-loop for effectively the whole control interval and
  one worker runs an approximately 50% duty cycle

#### Scenario: User-overridden worker count

- **WHEN** the user sets an explicit worker count `W` greater than zero
- **THEN** exactly `W` workers are started
- **AND** each runs a duty cycle of `target / W`, clamped to the range `[0, 1]`

### Requirement: Load accuracy

For any 1-second window beginning after a ramp-up period of at most 2 seconds and
ending at least 1 second before shutdown, the process's average CPU utilisation
SHALL be within the configured tolerance of the resolved target, expressed in
busy cores.

#### Scenario: Default tolerance

- **WHEN** the user does not configure a tolerance
- **THEN** the applied tolerance is +/- 0.1 busy cores

#### Scenario: Measurement window excludes ramp-up and shutdown

- **WHEN** utilisation is sampled during the first 2 seconds or the final 1
  second of the run
- **THEN** those samples are not subject to the accuracy requirement

### Requirement: Load stops with the run

When the run ends for any reason — target runtime reached, signal received, or
watchdog fired — every load-generating worker SHALL stop busy-looping within 100
milliseconds so the process does not consume CPU during shutdown.

#### Scenario: Workers stop on normal completion

- **WHEN** the resolved runtime elapses
- **THEN** all workers cease busy-looping within 100 ms
- **AND** the process's CPU utilisation drops toward zero before it exits

#### Scenario: Workers stop on signal

- **WHEN** a SIGINT or SIGTERM is received mid-run
- **THEN** all workers cease busy-looping within 100 ms as part of shutdown

### Requirement: Observed CPU reporting

The utility SHALL measure its own average CPU utilisation over the run and SHALL
include the observed value, in busy cores, in the final log record whenever at
least one worker ran.

#### Scenario: Reported alongside the target

- **WHEN** the run had a non-zero resolved CPU target
- **THEN** the final log record contains both the resolved target and the
  observed average utilisation
