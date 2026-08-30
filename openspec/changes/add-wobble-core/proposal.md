# Change: add-wobble-core

## Status

Proposed — 2026-08-30. No implementation exists yet; this change delivers the
first working binary.

## Why

The repository is an empty Go project. The four capability specs
(`execution-duration`, `cpu-load`, `exit-outcome`, `cli`) describe the intended
behaviour but nothing satisfies them. This change scaffolds the module and
implements every requirement in those specs so `wobble` can be built, tested, and
used as a workload simulator.

## What changes

- Introduce the Go module, `main` entrypoint, and internal package layout
  (see `design.md`).
- Implement configuration parsing and validation for the full flag / `WOBBLE_*`
  table, including the reserved-exit-code checks. — *cli*
- Implement single-RNG seeding with the fixed draw order
  (runtime → CPU → outcome) and seed logging. — *cli*
- Implement runtime sampling, `--max-duration` clamping, and the independent
  watchdog force-exit. — *execution-duration*
- Implement the duty-cycle CPU load workers, worker-count derivation,
  `GOMAXPROCS` clamping, self-measured CPU, and 100 ms stop-on-shutdown. —
  *cpu-load*
- Implement the Bernoulli success/failure draw and its mapping onto the exit
  code, including override by signal / watchdog. — *exit-outcome*
- Implement `slog`-based structured logging (`text` / `json`) with the startup
  summary and final records. — *cli*
- Implement SIGINT / SIGTERM handling with grace period and second-signal
  fast path. — *cli*
- Add unit and integration tests, one per scenario in the capability specs.
- Update the top-level `README.md` with usage and examples.

## Acceptance criteria

The base specs under `openspec/specs/` are the contract for this change. It is
complete when every `#### Scenario:` across all four capabilities has a passing
test. This change introduces no behavioural delta beyond what those specs already
state; on merge the specs stay as-is (they move from "described" to
"implemented").

## Impact

- Affected specs: `execution-duration`, `cpu-load`, `exit-outcome`, `cli`
  (all implemented, none modified).
- Affected code: new module — `main.go`, `internal/**`, `*_test.go`,
  `go.mod`, top-level `README.md`.
- Risk: CPU-load accuracy and watchdog timing are inherently machine- and
  scheduler-sensitive; mitigations are in `design.md` (generous tolerances,
  timing assertions with slack, `-short` skips for load tests in CI).
