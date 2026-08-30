# Tasks: add-wobble-core

Each task names the capability and scenario(s) it satisfies. Check off only when
the corresponding test passes.

## 1. Module scaffold

- [ ] `go mod init <module-path>` (decide path per `design.md` open question)
- [ ] Add `internal/` package skeletons: `config`, `rng`, `run`, `load`,
      `outcome`, `logx`
- [ ] `main.go`: parse → `run.Execute` → `os.Exit`; handle `--help` / `--version`
      → stdout, exit `0` — *cli: Help and version short-circuit*
- [ ] `make` / task targets: `build`, `test`, `test-short`, `lint`

## 2. Configuration (cli)

- [ ] Define `config.Config` with every field from the option table + defaults —
      *cli: Recognised options*
- [ ] `Parse(args []string, getenv func(string) string)` merging env then flags,
      flag wins — *cli: Flag overrides environment*
- [ ] `min..max` / scalar parser for durations and for floats, shared —
      *execution-duration: Ranged / Degenerate range*, *cpu-load: Ranged CPU target*
- [ ] Validation covering every bullet in *cli: Rejected configurations*,
      returning an error that names the option → exit `2`
- [ ] Reject `--failure-code` ∈ {2,124,130,143} — *exit-outcome: Reserved
      failure code rejected*, *cli: Failure code cannot shadow a reserved code*
- [ ] Confirm no RNG draw / no workers on any validation failure — *cli: Input
      validation* preamble

## 3. Seeding & RNG (cli)

- [ ] `rng.NewSource(seed uint64)` — single generator per run
- [ ] Generate seed from `crypto/rand` when unset — *cli: Seed generated and
      logged when unset*
- [ ] `DrawRuntime` / `DrawCPU` / `DrawOutcome` consuming draws only in the fixed
      order, and only when the input is a range — *cli: Deterministic seeding and
      draw order*
- [ ] Reproducibility test: same `(seed, config)` ⇒ same runtime, CPU, outcome —
      *cli: Reproducing a run*, *execution-duration: Ranged target runtime*,
      *cpu-load: Ranged CPU target*, *exit-outcome: Reproducible outcome*

## 4. Execution duration

- [ ] Resolve runtime at startup: fixed, range draw, or degenerate range —
      *execution-duration: Fixed / Ranged / Degenerate range*
- [ ] Zero runtime ⇒ no workers, still return outcome code — *execution-duration:
      Zero target runtime*
- [ ] Apply `--max-duration` default when unset — *execution-duration: Default
      maximum when unset*
- [ ] Clamp effective runtime to max + warn log — *execution-duration: Target
      runtime exceeds the maximum*
- [ ] Normal run timer (`time.After` / context deadline) triggers graceful
      shutdown; clean shutdown at ceiling keeps the outcome code —
      *execution-duration: Target runtime below the maximum / exceeds the maximum*
- [ ] Watchdog `AfterFunc(max + grace)` → final record + `os.Exit(124)`;
      `Stop()` on the normal path — *execution-duration: Watchdog termination*
      (both scenarios)
- [ ] Record elapsed runtime + terminal reason on all paths —
      *execution-duration: Elapsed time reporting*

## 5. CPU load

- [ ] `load.Pool` with `W` workers; derive `W = max(1, ceil(target))` or use
      `--workers` — *cpu-load: Worker count derived from the target /
      User-overridden worker count*
- [ ] Clamp target to `GOMAXPROCS` + warn — *cpu-load: Target exceeds available
      parallelism*
- [ ] Zero target ⇒ no busy-loop workers — *cpu-load: Zero CPU target*
- [ ] Duty-cycle loop (100 ms interval), per-worker `d = clamp(target/W, 0, 1)` —
      *cpu-load: Fractional-core target / Multi-core target*
- [ ] Context cancel stops every worker within 100 ms — *cpu-load: Load stops
      with the run* (both scenarios)
- [ ] Self CPU measurement via `Getrusage(RUSAGE_SELF)` deltas over a window
      excluding first 2 s / last 1 s — *cpu-load: Load accuracy*, *Measurement
      window excludes ramp-up and shutdown*
- [ ] Include `observed_cpu_cores` in final record when `W > 0` — *cpu-load:
      Observed CPU reporting*
- [ ] Accuracy test (`--cpu 0.5`, 4 s, observed ∈ [0.4, 0.6]), skipped under
      `-short` — *cpu-load: Default tolerance*

## 6. Exit outcome

- [ ] `DrawOutcome(p)` — single Bernoulli trial — *exit-outcome: Success rate
      draw*, *Typical rate* (frequency test)
- [ ] `p = 0` ⇒ always failure, `p = 1` ⇒ always success — *exit-outcome: Always
      fail / Always succeed*
- [ ] Default success rate applied when unset — *exit-outcome: Default rate*
- [ ] Draw at startup after runtime + CPU draws, held for the run —
      *exit-outcome: Decision timing*
- [ ] `ExitCode(cfg, outcome)` ⇒ `0` or `cfg.FailureCode` on normal completion —
      *exit-outcome: Successful / Failed completion*
- [ ] Custom `--failure-code 7` respected — *exit-outcome: Custom failure code*
- [ ] Signal / watchdog termination overrides the outcome code —
      *exit-outcome: Outcome overridden by abnormal termination*, *Signal during
      a run that had decided success*

## 7. Signals (cli)

- [ ] `signal.NotifyContext` for SIGINT/SIGTERM cancels the run context —
      *cli: Graceful stop on SIGTERM*
- [ ] `Execute` returns `130` (SIGINT) / `143` (SIGTERM), overriding outcome —
      *cli: Signal after the outcome would have been success*
- [ ] Second signal during shutdown ⇒ immediate `os.Exit(code)` — *cli:
      Impatient operator*
- [ ] Final record terminal reason `signal` on this path

## 8. Logging (cli)

- [ ] `logx.New(format, verbose)` → `slog` text/json handler on stderr;
      reject other formats in validation — *cli: Recognised options*
- [ ] Startup summary record with the required fields; `outcome` only when
      verbose — *cli: Startup summary log record*, *Non-verbose startup*,
      *exit-outcome: Outcome not revealed early by default / revealed under
      verbose*
- [ ] Final record with required fields on completed / signal / watchdog paths;
      not emitted for `--help` / `--version` / usage errors — *cli: Final log
      record* (both scenarios)
- [ ] stdout carries only `--help` / `--version` text

## 9. Exit-code table (cli)

- [ ] Named constants `0 / 2 / 124 / 130 / 143` + `cfg.FailureCode`, referenced
      everywhere — *cli: Exit-code table*, *Codes are stable identifiers*
- [ ] Integration test asserting each row's exit code end-to-end

## 10. Docs & release

- [ ] Replace top-level `README.md` stub with purpose, install, flag table,
      exit-code table, and 3–4 worked examples (scheduler retry, autoscaler,
      timeout wrapper, chaos)
- [ ] `wobble --version` wired to build-time ldflags
- [ ] CI: `go test ./...` (with and without `-short`), `go vet`, `staticcheck`

## Definition of done

- [ ] Every `#### Scenario:` in the four capability specs has a passing test
- [ ] `go build ./...` and `go test ./...` green on Linux and macOS
- [ ] `openspec/specs/` unchanged (no behavioural delta); this change archived
      under `openspec/changes/archive/`
