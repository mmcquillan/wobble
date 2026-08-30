# Design: add-wobble-core

## Module & layout

Module path: `github.com/wobble` (adjust before first `go mod init` if a
different path is preferred). Go 1.22+ (uses `log/slog`, `math/rand/v2`).

```
main.go                     flag/env parse -> Config -> run.Execute -> os.Exit(code)
internal/config/            Config struct, Parse(args, getenv) (Config, error), validation
internal/config/rangeval.go "min..max" / scalar parsing for durations and floats
internal/rng/               NewSource(seed) plus DrawRuntime / DrawCPU / DrawOutcome in fixed order
internal/run/               Execute(cfg): sample, log startup, start load, wait, shutdown, log final
internal/run/watchdog.go    time.AfterFunc(maxDuration+grace) -> os.Exit(124)
internal/load/              Pool: worker goroutines, duty-cycle loop, self CPU measurement
internal/outcome/           Outcome (success|failure) and ExitCode(cfg, outcome)
internal/logx/              slog handler selection (text|json), StartupRecord, FinalRecord helpers
```

Keep `main.go` tiny: it only wires parse → execute → exit and never calls
`os.Exit` itself except with the code `run.Execute` returns (plus `2` for a parse
error and `0` for `--help`/`--version`).

## Exit-code flow

`run.Execute` returns an `int` for the normal and signal paths. The watchdog and
the "second signal" fast path call `os.Exit` directly because they must win over
any in-flight shutdown. Codes are defined once as named constants in
`internal/run` (or a small `exit` package) and referenced everywhere:

```
CodeSuccess = 0
CodeUsage   = 2
CodeWatchdog = 124
CodeSIGINT  = 130
CodeSIGTERM = 143
// failure code is cfg.FailureCode (default 1), validated != any reserved code
```

## RNG and determinism

- One `*rand.Rand` (from `math/rand/v2`, `rand.NewChaCha8` or `rand.New(rand.NewPCG(seed, 0))`)
  created in `rng.NewSource(seed)`.
- Draw order is fixed and centralised so tests can assert reproducibility:
  1. `DrawRuntime` — only consumes a draw when `--duration` is a range.
  2. `DrawCPU` — only consumes a draw when `--cpu` is a range.
  3. `DrawOutcome` — always consumes exactly one `Float64()` compared to `p`.
- Unset seed: read 8 bytes from `crypto/rand`, fold to `uint64`, log it.
- Worker scheduling uses wall-clock time, not the seeded RNG, so it is explicitly
  outside the determinism guarantee (matches the cli spec scenario).

## CPU load model

- Effective target `C` = clamp(resolved target, 0, GOMAXPROCS).
- Worker count `W` = `--workers` if > 0, else `max(1, ceil(C))` when `C > 0`,
  else `0`.
- Per-worker duty `d = clamp(C / W, 0, 1)`.
- Control interval `I = 100ms`. Each worker loop:
  `busy for d*I` (tight integer arithmetic loop, `runtime.Gosched()`-free),
  then `sleep (1-d)*I`, re-checking a `context.Context` each interval so it
  stops within one interval (≤ 100 ms) on cancel.
- Ramp-up: first ~2 intervals may be off-target; the accuracy requirement
  already excludes the first 2 s.
- Optional refinement (not required by spec, keep behind a constant): a slow
  outer correction that nudges `d` based on measured vs target CPU. Start
  open-loop; add only if tests show systematic bias.

## Self CPU measurement

- Sample `runtime` is not enough; use OS-reported process CPU time:
  - Linux/macOS: `syscall.Getrusage(RUSAGE_SELF)` deltas (`Utime + Stime`)
    over a wall-clock delta → busy cores.
  - Take a baseline right after workers start (post ramp), another near the end
    (pre shutdown), divide CPU-seconds by wall-seconds.
- Report `observed_cpu_cores` in the final record when `W > 0`.

## Watchdog

- `time.AfterFunc(cfg.MaxDuration + cfg.Grace, func(){ logx.Final(..., "watchdog", 124); os.Exit(124) })`.
- Normal completion path cancels it via `timer.Stop()` before returning.
- The normal run timer is a separate `time.After(effectiveRuntime)` (or context
  deadline) — reaching it triggers graceful shutdown, not the watchdog.

## Signals

- `signal.NotifyContext` for SIGINT/SIGTERM cancels the run context → workers
  stop, shutdown proceeds, `Execute` returns 130/143 based on which signal.
- A second goroutine does `signal.Notify` on a buffered channel; on the *second*
  delivery it calls `os.Exit(code)` immediately.
- Signal path sets terminal reason `signal`; the decided outcome is still logged
  but does not affect the code.

## Logging

- `logx.New(format, verbose)` returns an `*slog.Logger` over `slog.NewTextHandler`
  or `slog.NewJSONHandler` writing to `os.Stderr`.
- Startup record (info): `seed, runtime, max_duration, grace, cpu_target,
  workers, success_rate` and, only if verbose, `outcome`.
- Final record (info, or error when code ∉ {0}): `elapsed, cpu_target,
  observed_cpu_cores?, outcome, terminal_reason, exit_code`.
- stdout is reserved for `--help` / `--version` text only.

## Testing strategy

- `internal/config`: table-driven tests for every rejected-configuration bullet
  and the flag-over-env precedence scenario.
- `internal/rng`: reproducibility (same seed+config ⇒ same triple), draw-order,
  and "no draw consumed when scalar" tests using a recording source.
- `internal/outcome`: `p=0`, `p=1`, and a large-N frequency test with tolerance.
- `internal/run`: integration tests that build the binary (`go test` +
  `exec.Command`) and assert exit codes / log records for: completed-success,
  completed-failure, zero-duration, clamp-and-warn, SIGINT, SIGTERM,
  double-SIGINT, watchdog (`--max-duration 200ms --grace 100ms` with a wedged
  hook via a test-only env var), usage errors.
- `internal/load`: accuracy test guarded by `testing.Short()` — run `--cpu 0.5`
  for 4 s, assert observed ∈ [0.4, 0.6]; skipped in `-short` / constrained CI.
- All timing assertions carry ≥ 50 % slack to stay non-flaky.

## Open questions

- Whether `--cpu` should also accept a `N%` form (e.g. `150%` = 1.5 cores).
  Deferred; `--cpu 1.5` covers it.
- Windows support for self-CPU measurement (`GetProcessTimes`) — deferred;
  document as best-effort / omit `observed_cpu_cores` there.
