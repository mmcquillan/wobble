# Project: wobble

## Purpose

`wobble` is a small command-line workload simulator. A single run:

1. Executes for a **variable amount of wall-clock time**, drawn from a fixed value
   or a range, and always bounded by a hard **maximum allowable duration**.
2. Consumes a **variable amount of CPU**, drawn from a fixed value or a range,
   held roughly constant for the life of the run.
3. Exits with a **success or failure exit code**, chosen per run by a configurable
   **success probability**.

It exists to exercise the systems that watch other processes: job schedulers,
Kubernetes `Job`/`CronJob` back-off, autoscalers, retry/circuit-breaker logic,
CI pipelines, timeout wrappers, and monitoring/alerting rules. Given a seed, a run
is fully reproducible.

## Non-goals

- Doing any useful work. CPU cycles are burned, not applied.
- Simulating memory pressure, I/O load, or network behaviour (possible future
  capabilities, out of scope now).
- Being a benchmark. Load targets are approximate, not precise.

## Tech stack

- Go (single statically-linked binary, standard library only where practical).
- No configuration files; all input via flags and `WOBBLE_*` environment
  variables.
- Structured logging to stderr (`text` or `json`); nothing on stdout except
  `--help` / `--version`.

## Conventions

- Specs live in `openspec/specs/<capability>/spec.md`, one capability per
  directory, written as `### Requirement:` blocks each followed by one or more
  `#### Scenario:` blocks using `WHEN` / `THEN` / `AND` bullets.
- Keywords **SHALL**, **SHALL NOT**, **MAY** are normative (RFC 2119 sense).
- Changes to behaviour are proposed under `openspec/changes/<change-id>/` before
  the capability specs are edited.
- The authoritative exit-code table is in `openspec/specs/cli/spec.md`; other
  specs reference codes by name, not number.

## Capabilities

| Capability | Directory | Concern |
|------------|-----------|---------|
| Execution duration | `specs/execution-duration/` | How long a run lasts; the hard ceiling and watchdog |
| CPU load | `specs/cpu-load/` | How much CPU a run burns and how steadily |
| Exit outcome | `specs/exit-outcome/` | The success/failure draw and its exit code |
| CLI | `specs/cli/` | Config surface, validation, seeding, signals, logging, exit-code table |
