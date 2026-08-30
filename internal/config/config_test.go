package config

import (
	"strings"
	"testing"
	"time"
)

func envFunc(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

var noEnv = envFunc(nil)

func TestDefaults(t *testing.T) {
	res, err := Parse(nil, noEnv, "test")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := res.Config
	if c == nil {
		t.Fatal("nil Config")
	}
	if c.MaxDuration != DefaultMaxDuration || c.Grace != DefaultGrace {
		t.Errorf("durations = %v/%v", c.MaxDuration, c.Grace)
	}
	if c.SuccessRate != 1.0 || c.FailureCode != 1 || c.Tolerance != 0.1 {
		t.Errorf("rate/code/tol = %v/%v/%v", c.SuccessRate, c.FailureCode, c.Tolerance)
	}
	if c.LogFormat != "text" || c.Verbose {
		t.Errorf("logformat/verbose = %q/%v", c.LogFormat, c.Verbose)
	}
	if c.Duration.IsRange || c.Duration.Min != 0 {
		t.Errorf("duration = %+v", c.Duration)
	}
	if c.SeedSet {
		t.Error("SeedSet should be false when --seed is unset")
	}
}

func TestHelpAndVersion(t *testing.T) {
	res, err := Parse([]string{"--help"}, noEnv, "9.9")
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	if res.Config != nil || !strings.Contains(res.Usage, "Usage: wobble") {
		t.Errorf("--help result = %+v", res)
	}

	res, err = Parse([]string{"--version"}, noEnv, "9.9")
	if err != nil {
		t.Fatalf("--version: %v", err)
	}
	if res.Version != "wobble 9.9\n" {
		t.Errorf("--version = %q", res.Version)
	}
	if !res.EarlyExit() {
		t.Error("EarlyExit should be true")
	}
}

func TestFlagOverridesEnv(t *testing.T) {
	env := envFunc(map[string]string{"WOBBLE_CPU": "1.0"})
	res, err := Parse([]string{"--cpu", "2.0"}, env, "t")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Config.CPU.Min != 2.0 {
		t.Errorf("cpu = %v, want 2.0 (flag wins over env)", res.Config.CPU.Min)
	}
}

func TestEnvUsedWithoutFlag(t *testing.T) {
	env := envFunc(map[string]string{
		"WOBBLE_SUCCESS_RATE": "0.5",
		"WOBBLE_VERBOSE":      "true",
		"WOBBLE_SEED":         "77",
	})
	res, err := Parse(nil, env, "t")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := res.Config
	if c.SuccessRate != 0.5 || !c.Verbose || c.Seed != 77 || !c.SeedSet {
		t.Errorf("env not applied: %+v", c)
	}
}

func TestDurationRange(t *testing.T) {
	res, err := Parse([]string{"--duration", "1s..3s"}, noEnv, "t")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	d := res.Config.Duration
	if !d.IsRange || d.Min != time.Second || d.Max != 3*time.Second {
		t.Errorf("range = %+v", d)
	}
}

func TestDegenerateRangeCollapses(t *testing.T) {
	res, err := Parse([]string{"--duration", "2s..2s", "--cpu", "1.5..1.5"}, noEnv, "t")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.Config.Duration.IsRange {
		t.Error("duration 2s..2s should collapse to fixed (no RNG draw)")
	}
	if res.Config.CPU.IsRange {
		t.Error("cpu 1.5..1.5 should collapse to fixed")
	}
}

func TestSeedParsed(t *testing.T) {
	res, err := Parse([]string{"--seed", "-42"}, noEnv, "t")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var want int64 = -42
	if !res.Config.SeedSet || res.Config.Seed != uint64(want) {
		t.Errorf("seed = %d set=%v", res.Config.Seed, res.Config.SeedSet)
	}
}

func TestRejectedConfigurations(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string // substring the error must name
	}{
		{"negative duration", []string{"--duration=-5s"}, "duration"},
		{"range min>max", []string{"--duration", "9s..3s"}, "duration"},
		{"max-duration zero", []string{"--max-duration", "0s"}, "max-duration"},
		{"grace negative", []string{"--grace=-1s"}, "grace"},
		{"cpu negative", []string{"--cpu=-1"}, "cpu"},
		{"cpu range negative", []string{"--cpu", "-1..2"}, "cpu"},
		{"success-rate high", []string{"--success-rate", "1.5"}, "success-rate"},
		{"success-rate low", []string{"--success-rate", "-0.1"}, "success-rate"},
		{"failure-code zero", []string{"--failure-code", "0"}, "failure-code"},
		{"failure-code 256", []string{"--failure-code", "256"}, "failure-code"},
		{"failure-code reserved 2", []string{"--failure-code", "2"}, "reserved"},
		{"failure-code reserved 124", []string{"--failure-code", "124"}, "reserved"},
		{"failure-code reserved 130", []string{"--failure-code", "130"}, "reserved"},
		{"failure-code reserved 143", []string{"--failure-code", "143"}, "reserved"},
		{"workers negative", []string{"--workers=-1"}, "workers"},
		{"tolerance zero", []string{"--tolerance", "0"}, "tolerance"},
		{"tolerance negative", []string{"--tolerance=-1"}, "tolerance"},
		{"log-format bad", []string{"--log-format", "yaml"}, "log-format"},
		{"unparseable duration", []string{"--duration", "soon"}, "duration"},
		{"unparseable seed", []string{"--seed", "abc"}, "seed"},
		{"unknown flag", []string{"--nope"}, "nope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Parse(tc.args, noEnv, "t")
			if err == nil {
				t.Fatalf("expected error, got Config %+v", res.Config)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not name %q", err.Error(), tc.want)
			}
		})
	}
}

func TestBadEnvVerbose(t *testing.T) {
	env := envFunc(map[string]string{"WOBBLE_VERBOSE": "maybe"})
	if _, err := Parse(nil, env, "t"); err == nil {
		t.Fatal("expected error for WOBBLE_VERBOSE=maybe")
	}
}
