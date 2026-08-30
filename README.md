# wobble

A small workload simulator.


## Disclaimer

This was wholly produced via Claude (Sonnet 4.6) with the following prompts:

```
We are going to create an openspec style specification for this small go utility that needs to execute for a variable amount of time (with a max allowable time), use a variable amount of CPU, and quit with a percent of success versus failure exit code. Can you write this for me?

Implement wobble in go based on the available specs in the repo
```


## Overview

1. executes for a **variable amount of wall-clock time** (fixed or sampled from a
   range), always bounded by a hard **maximum duration**;
2. consumes a **variable amount of CPU** (fixed or sampled from a range), held
   roughly constant for the run;
3. exits with a **success or failure exit code**, chosen per run by a configurable
   success probability.

It exists to exercise the things that watch other processes: job schedulers,
Kubernetes `Job`/`CronJob` back-off, autoscalers, retry and circuit-breaker
logic, CI pipelines, `timeout(1)` wrappers, and monitoring/alerting. Given a
`--seed`, a run is fully reproducible.

The behaviour is specified in [`openspec/`](openspec/).

## Build

```bash
go build -o wobble .
```

```bash
go build -ldflags "-X main.version=$(git describe --tags --always)" -o wobble .
```

## Usage

```
wobble [flags]
```

| Flag | Env | Meaning | Default |
|------|-----|---------|---------|
| `--duration` | `WOBBLE_DURATION` | Target runtime: Go duration, or `MIN..MAX` | `0s` |
| `--max-duration` | `WOBBLE_MAX_DURATION` | Absolute wall-clock ceiling (`> 0`) | `5m` |
| `--grace` | `WOBBLE_GRACE` | Shutdown grace period before the watchdog fires | `2s` |
| `--cpu` | `WOBBLE_CPU` | CPU target in busy cores: float `>= 0`, or `MIN..MAX` | `0` |
| `--workers` | `WOBBLE_WORKERS` | Explicit load-worker count; `0` derives it from `--cpu` | `0` |
| `--tolerance` | `WOBBLE_TOLERANCE` | CPU accuracy tolerance in busy cores (`> 0`) | `0.1` |
| `--success-rate` | `WOBBLE_SUCCESS_RATE` | Probability of exit `0`, in `[0,1]` | `1.0` |
| `--failure-code` | `WOBBLE_FAILURE_CODE` | Exit code on a failure outcome (`1..255`, not reserved) | `1` |
| `--seed` | `WOBBLE_SEED` | `int64` RNG seed; random (and logged) when unset | random |
| `--log-format` | `WOBBLE_LOG_FORMAT` | `text` or `json` (structured logs on stderr) | `text` |
| `--verbose`, `-v` | `WOBBLE_VERBOSE` | Extra logging, incl. the decided outcome at startup | `false` |
| `--version` | — | Print version and exit `0` | |
| `--help`, `-h` | — | Print help and exit `0` | |

A flag always wins over its environment variable. `MIN..MAX` ranges are sampled
once at startup from the seeded RNG, in the order runtime → CPU → outcome, so
`(seed, config)` fully determines the run.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | completed; outcome = success |
| `N` | completed; outcome = failure (`N` = `--failure-code`, default `1`) |
| `2` | invalid configuration / usage error (no work performed) |
| `124` | watchdog: process still alive one grace period after `--max-duration` |
| `130` | terminated by SIGINT |
| `143` | terminated by SIGTERM |

SIGINT/SIGTERM stop the load, emit the final log record, and override the
success/failure code. A second signal exits immediately.

## Examples

Run for 2–5 minutes at 1.5 cores, failing 10% of the time — a flaky batch job:

```bash
wobble --duration 2m..5m --cpu 1.5 --success-rate 0.9
```

Reproduce a specific run exactly:

```bash
wobble --duration 2m..5m --cpu 1.5 --success-rate 0.9 --seed 8571234
```

Deterministic failure after ~10s, custom code — test a scheduler's back-off:

```bash
wobble --duration 10s --success-rate 0 --failure-code 42
```

Saturate 4 cores for 30s, JSON logs — feed an autoscaler:

```bash
wobble --duration 30s --cpu 4 --log-format json
```

Guaranteed-bounded chaos: never exceed 90s wall-clock even if wedged:

```bash
wobble --duration 30s..10m --max-duration 90s --grace 5s --cpu 0.5..3
```

## Development

```bash
go test ./...            # full suite
go test -short ./...      # skips the multi-second CPU-accuracy test
go test -race ./...
go vet ./...
```
