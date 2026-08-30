// Package config parses and validates wobble's configuration from command-line
// flags and WOBBLE_* environment variables. Flags win over the environment.
// See openspec/specs/cli/spec.md.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"crypto/rand"
	"encoding/binary"

	"github.com/wobble/internal/exitcode"
)

// Defaults for every tunable, per the CLI spec's option table.
const (
	DefaultMaxDuration = 5 * time.Minute
	DefaultGrace       = 2 * time.Second
	DefaultTolerance   = 0.1
	DefaultSuccessRate = 1.0
	DefaultFailureCode = 1
	DefaultLogFormat   = "text"
)

// Config is fully validated configuration for one run.
type Config struct {
	Duration    DurationSpec
	MaxDuration time.Duration
	Grace       time.Duration
	CPU         CPUSpec
	Workers     int
	Tolerance   float64
	SuccessRate float64
	FailureCode int
	Seed        uint64
	SeedSet     bool // false => Seed was generated and should be logged for replay
	LogFormat   string
	Verbose     bool
}

// Result is the outcome of Parse. Exactly one of the fields is populated:
// Config for a normal run, or Usage / Version for an early exit that prints to
// stdout and exits 0.
type Result struct {
	Config  *Config
	Usage   string
	Version string
}

// EarlyExit reports whether Parse handled --help or --version.
func (r *Result) EarlyExit() bool { return r.Usage != "" || r.Version != "" }

// Parse reads args and the environment (via getenv) into a Result. A non-nil
// error means a usage error: the caller should print it to stderr and exit with
// exitcode.Usage. No randomness is drawn and no work is started on that path.
func Parse(args []string, getenv func(string) string, version string) (*Result, error) {
	fs := flag.NewFlagSet("wobble", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	var (
		fDuration  = fs.String("duration", "", "")
		fMax       = fs.String("max-duration", "", "")
		fGrace     = fs.String("grace", "", "")
		fCPU       = fs.String("cpu", "", "")
		fWorkers   = fs.String("workers", "", "")
		fTolerance = fs.String("tolerance", "", "")
		fSuccess   = fs.String("success-rate", "", "")
		fFailure   = fs.String("failure-code", "", "")
		fSeed      = fs.String("seed", "", "")
		fLogFormat = fs.String("log-format", "", "")
		fVerbose   = fs.Bool("verbose", false, "")
		fVerboseSh = fs.Bool("v", false, "")
		fVersion   = fs.Bool("version", false, "")
	)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return &Result{Usage: usageText}, nil
		}
		return nil, err
	}
	if *fVersion {
		return &Result{Version: fmt.Sprintf("wobble %s\n", version)}, nil
	}

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	// pick returns the effective raw string for an option: the flag value if the
	// flag was passed, else the environment value if non-empty, else ("", false).
	pick := func(flagName, envName string, flagVal *string) (string, bool) {
		if set[flagName] {
			return *flagVal, true
		}
		if v := strings.TrimSpace(getenv(envName)); v != "" {
			return v, true
		}
		return "", false
	}

	cfg := &Config{
		MaxDuration: DefaultMaxDuration,
		Grace:       DefaultGrace,
		Tolerance:   DefaultTolerance,
		SuccessRate: DefaultSuccessRate,
		FailureCode: DefaultFailureCode,
		LogFormat:   DefaultLogFormat,
	}

	if raw, ok := pick("duration", "WOBBLE_DURATION", fDuration); ok {
		ds, err := parseDurationSpec(raw)
		if err != nil {
			return nil, fmt.Errorf("--duration: %w", err)
		}
		cfg.Duration = ds
	}

	if raw, ok := pick("max-duration", "WOBBLE_MAX_DURATION", fMax); ok {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("--max-duration: %w", err)
		}
		cfg.MaxDuration = d
	}
	if cfg.MaxDuration <= 0 {
		return nil, fmt.Errorf("--max-duration: must be > 0")
	}

	if raw, ok := pick("grace", "WOBBLE_GRACE", fGrace); ok {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("--grace: %w", err)
		}
		cfg.Grace = d
	}
	if cfg.Grace <= 0 {
		return nil, fmt.Errorf("--grace: must be > 0")
	}

	if raw, ok := pick("cpu", "WOBBLE_CPU", fCPU); ok {
		cs, err := parseCPUSpec(raw)
		if err != nil {
			return nil, fmt.Errorf("--cpu: %w", err)
		}
		cfg.CPU = cs
	}

	if raw, ok := pick("workers", "WOBBLE_WORKERS", fWorkers); ok {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("--workers: %w", err)
		}
		if n < 0 {
			return nil, fmt.Errorf("--workers: must be >= 0")
		}
		cfg.Workers = n
	}

	if raw, ok := pick("tolerance", "WOBBLE_TOLERANCE", fTolerance); ok {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("--tolerance: %w", err)
		}
		cfg.Tolerance = f
	}
	if cfg.Tolerance <= 0 {
		return nil, fmt.Errorf("--tolerance: must be > 0")
	}

	if raw, ok := pick("success-rate", "WOBBLE_SUCCESS_RATE", fSuccess); ok {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("--success-rate: %w", err)
		}
		cfg.SuccessRate = f
	}
	if cfg.SuccessRate < 0 || cfg.SuccessRate > 1 {
		return nil, fmt.Errorf("--success-rate: must be in [0, 1]")
	}

	if raw, ok := pick("failure-code", "WOBBLE_FAILURE_CODE", fFailure); ok {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("--failure-code: %w", err)
		}
		cfg.FailureCode = n
	}
	if cfg.FailureCode < 1 || cfg.FailureCode > 255 {
		return nil, fmt.Errorf("--failure-code: must be in 1..255")
	}
	if exitcode.Reserved(cfg.FailureCode) {
		return nil, fmt.Errorf("--failure-code: %d is reserved (2, 124, 130, 143)", cfg.FailureCode)
	}

	seedRaw, seedGiven := pick("seed", "WOBBLE_SEED", fSeed)
	if seedGiven {
		n, err := strconv.ParseInt(seedRaw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("--seed: %w", err)
		}
		cfg.Seed = uint64(n)
		cfg.SeedSet = true
	}

	if raw, ok := pick("log-format", "WOBBLE_LOG_FORMAT", fLogFormat); ok {
		cfg.LogFormat = raw
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return nil, fmt.Errorf("--log-format: must be \"text\" or \"json\"")
	}

	cfg.Verbose = *fVerbose || *fVerboseSh
	if !cfg.Verbose {
		if v := strings.TrimSpace(getenv("WOBBLE_VERBOSE")); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("WOBBLE_VERBOSE: %w", err)
			}
			cfg.Verbose = b
		}
	}

	// Only now that every value is valid do we draw a random seed (never from the
	// run's seeded generator, so the "no RNG drawn on error" guarantee holds).
	if !cfg.SeedSet {
		cfg.Seed = randomSeed()
	}

	return &Result{Config: cfg}, nil
}

func randomSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint64(time.Now().UnixNano())
	}
	return binary.LittleEndian.Uint64(b[:])
}

const usageText = `Usage: wobble [flags]

wobble runs for a variable amount of time, burns a variable amount of CPU, and
exits success or failure according to a configured probability.

Flags:
  --duration VALUE        Target runtime: Go duration, or MIN..MAX range (default 0s)
  --max-duration VALUE    Absolute wall-clock ceiling, > 0 (default 5m0s)
  --grace VALUE           Shutdown grace period before the watchdog fires (default 2s)
  --cpu VALUE             CPU target in busy cores: float >= 0, or MIN..MAX range (default 0)
  --workers N             Explicit load-worker count; 0 derives it from --cpu (default 0)
  --tolerance F           CPU accuracy tolerance in busy cores, > 0 (default 0.1)
  --success-rate P        Probability of exit 0, in [0,1] (default 1.0)
  --failure-code N        Exit code on a failure outcome: 1..255, not 2/124/130/143 (default 1)
  --seed N                int64 RNG seed; random (and logged) when unset
  --log-format text|json  Structured log encoding on stderr (default text)
  --verbose, -v           Extra logging, including the decided outcome at startup
  --version               Print version and exit
  --help, -h              Print this help and exit

Every flag can also be set via WOBBLE_<FLAG> (dashes become underscores); the
flag wins over the environment.

Exit codes:
  0    completed; outcome = success
  N    completed; outcome = failure (N = --failure-code, default 1)
  2    invalid configuration / usage error
  124  watchdog: still alive one grace period after --max-duration
  130  terminated by SIGINT
  143  terminated by SIGTERM
`
