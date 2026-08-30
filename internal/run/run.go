// Package run orchestrates a single wobble run: resolve the random draws, log
// the startup summary, generate load for the resolved runtime, then shut down
// and report an exit code. The watchdog and the second-signal fast path call
// os.Exit directly because they must win over any in-flight shutdown.
package run

import (
	"context"
	"io"
	"math"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wobble/internal/config"
	"github.com/wobble/internal/exitcode"
	"github.com/wobble/internal/load"
	"github.com/wobble/internal/logx"
	"github.com/wobble/internal/outcome"
	"github.com/wobble/internal/rng"
)

// wedgeShutdownEnv, when set to a Go duration, makes shutdown sleep that long.
// This is a test-only hook for exercising the watchdog and is intentionally
// undocumented in --help.
const wedgeShutdownEnv = "WOBBLE_TEST_WEDGE_SHUTDOWN"

// Execute runs one wobble session and returns its process exit code. Structured
// logs are written to stderr.
func Execute(cfg *config.Config, stderr io.Writer) int {
	start := time.Now()
	logger := logx.New(stderr, cfg.LogFormat, cfg.Verbose)

	// Fixed draw order: runtime, then CPU target, then outcome.
	src := rng.New(cfg.Seed)
	resolvedRuntime := src.DrawRuntime(cfg.Duration.Min, cfg.Duration.Max, cfg.Duration.IsRange)
	resolvedCPU := src.DrawCPU(cfg.CPU.Min, cfg.CPU.Max, cfg.CPU.IsRange)
	decided := outcome.Decide(src.DrawOutcome(cfg.SuccessRate))

	// Clamp the runtime to the absolute maximum.
	effRuntime := resolvedRuntime
	if effRuntime > cfg.MaxDuration {
		logger.Warn("runtime clamped to max-duration",
			"resolved_runtime", resolvedRuntime, "max_duration", cfg.MaxDuration)
		effRuntime = cfg.MaxDuration
	}
	if effRuntime < 0 {
		effRuntime = 0
	}

	// Clamp the CPU target to available parallelism.
	maxProcs := float64(runtime.GOMAXPROCS(0))
	effCPU := resolvedCPU
	if effCPU > maxProcs {
		logger.Warn("cpu target clamped to available parallelism",
			"resolved_cpu", round3(resolvedCPU), "gomaxprocs", maxProcs)
		effCPU = maxProcs
	}

	duties := load.Duties(effCPU, cfg.Workers)
	if effRuntime == 0 {
		duties = nil // zero runtime => no workers
	}

	startFields := []any{
		"seed", cfg.Seed,
		"seed_generated", !cfg.SeedSet,
		"runtime", effRuntime,
		"max_duration", cfg.MaxDuration,
		"grace", cfg.Grace,
		"cpu_target", round3(effCPU),
		"workers", len(duties),
		"success_rate", cfg.SuccessRate,
	}
	if cfg.Verbose {
		startFields = append(startFields, "outcome", decided.String())
	}
	logger.Info("startup", startFields...)

	// Only one of {normal path, watchdog} may emit the final record / exit.
	var finalized atomic.Bool
	emitFinal := func(elapsed time.Duration, observed float64, haveObserved bool, reason string, code int) {
		fields := []any{"elapsed", elapsed, "cpu_target", round3(effCPU)}
		if haveObserved {
			fields = append(fields, "observed_cpu_cores", round3(observed))
		}
		fields = append(fields,
			"outcome", decided.String(),
			"terminal_reason", reason,
			"exit_code", code,
		)
		if code == exitcode.Success {
			logger.Info("final", fields...)
		} else {
			logger.Error("final", fields...)
		}
	}

	// Watchdog: fires one grace period after the absolute maximum, measured from
	// process start.
	watchdog := time.AfterFunc(cfg.MaxDuration+cfg.Grace, func() {
		if !finalized.CompareAndSwap(false, true) {
			return
		}
		emitFinal(time.Since(start), 0, false, "watchdog", exitcode.Watchdog)
		os.Exit(exitcode.Watchdog)
	})
	defer watchdog.Stop()

	// Signal handling: first signal starts shutdown; a second exits immediately.
	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	var sigNum atomic.Int32
	go func() {
		n := 0
		for s := range sigCh {
			n++
			sig, _ := s.(syscall.Signal)
			if n == 1 {
				sigNum.Store(int32(sig))
				cancelRun()
				continue
			}
			os.Exit(sigCodeOf(syscall.Signal(sigNum.Load())))
		}
	}()

	pool := load.Start(runCtx, duties)

	runTimer := time.NewTimer(effRuntime)
	select {
	case <-runTimer.C:
	case <-runCtx.Done():
		runTimer.Stop()
	}

	pool.Stop()
	observed, haveObserved := pool.ObservedCores()

	// Test-only: hold shutdown open long enough for the watchdog to win.
	if raw := os.Getenv(wedgeShutdownEnv); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			time.Sleep(d)
		}
	}

	reason := "completed"
	code := outcome.ExitCode(decided, cfg.FailureCode)
	if sig := syscall.Signal(sigNum.Load()); sig != 0 {
		reason = "signal"
		code = sigCodeOf(sig)
	}

	if !finalized.CompareAndSwap(false, true) {
		// Watchdog already fired and is force-exiting; wait for it.
		for {
			time.Sleep(time.Second)
		}
	}
	emitFinal(time.Since(start), observed, haveObserved, reason, code)
	return code
}

func sigCodeOf(s syscall.Signal) int {
	if s == syscall.SIGTERM {
		return exitcode.SIGTERM
	}
	return exitcode.SIGINT
}

func round3(f float64) float64 { return math.Round(f*1000) / 1000 }
