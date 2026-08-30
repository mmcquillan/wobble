package run_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

var bin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "wobble-bin")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(dir, "wobble")
	if out, err := exec.Command("go", "build", "-o", bin, "github.com/wobble").CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	code    int
	output  string
	elapsed time.Duration
}

func runWobble(t *testing.T, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	var buf strings.Builder
	cmd.Stdout, cmd.Stderr = &buf, &buf

	start := time.Now()
	err := cmd.Run()
	el := time.Since(start)

	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	return result{code: code, output: buf.String(), elapsed: el}
}

func startWobble(t *testing.T, env []string, args ...string) (*exec.Cmd, *strings.Builder) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	buf := &strings.Builder{}
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	return cmd, buf
}

func waitCode(t *testing.T, cmd *exec.Cmd) int {
	t.Helper()
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	t.Fatalf("wait: %v", err)
	return -1
}

func mustContain(t *testing.T, out, sub string) {
	t.Helper()
	if !strings.Contains(out, sub) {
		t.Errorf("output missing %q:\n%s", sub, out)
	}
}

func TestCompletedSuccess(t *testing.T) {
	r := runWobble(t, nil, "--duration", "20ms", "--success-rate", "1", "--seed", "1")
	if r.code != 0 {
		t.Fatalf("code = %d\n%s", r.code, r.output)
	}
	mustContain(t, r.output, "terminal_reason=completed")
	mustContain(t, r.output, "exit_code=0")
	mustContain(t, r.output, "msg=startup")
	mustContain(t, r.output, "msg=final")
}

func TestCompletedFailureDefaultCode(t *testing.T) {
	r := runWobble(t, nil, "--duration", "20ms", "--success-rate", "0", "--seed", "1")
	if r.code != 1 {
		t.Fatalf("code = %d, want 1\n%s", r.code, r.output)
	}
	mustContain(t, r.output, "outcome=failure")
	mustContain(t, r.output, "terminal_reason=completed")
}

func TestCompletedFailureCustomCode(t *testing.T) {
	r := runWobble(t, nil, "--duration", "20ms", "--success-rate", "0", "--failure-code", "7", "--seed", "1")
	if r.code != 7 {
		t.Fatalf("code = %d, want 7\n%s", r.code, r.output)
	}
}

func TestZeroDurationStillReturnsOutcome(t *testing.T) {
	r := runWobble(t, nil, "--duration", "0s", "--success-rate", "0", "--seed", "1")
	if r.code != 1 {
		t.Fatalf("code = %d, want 1\n%s", r.code, r.output)
	}
	mustContain(t, r.output, "workers=0")
	if r.elapsed > 2*time.Second {
		t.Errorf("zero-duration run took %v", r.elapsed)
	}
}

func TestRuntimeClampedToMax(t *testing.T) {
	r := runWobble(t, nil, "--duration", "10s", "--max-duration", "80ms", "--success-rate", "1", "--seed", "1")
	if r.code != 0 {
		t.Fatalf("code = %d\n%s", r.code, r.output)
	}
	mustContain(t, r.output, "clamped to max-duration")
	if r.elapsed > 2*time.Second {
		t.Errorf("clamped run took %v, expected ~80ms", r.elapsed)
	}
}

func TestVerboseRevealsOutcomeAtStartup(t *testing.T) {
	plain := runWobble(t, nil, "--duration", "10ms", "--seed", "1")
	startLine := firstLineWith(plain.output, "msg=startup")
	if strings.Contains(startLine, "outcome=") {
		t.Errorf("non-verbose startup leaked outcome: %s", startLine)
	}

	verbose := runWobble(t, nil, "--duration", "10ms", "--seed", "1", "--verbose")
	if !strings.Contains(firstLineWith(verbose.output, "msg=startup"), "outcome=") {
		t.Errorf("verbose startup missing outcome:\n%s", verbose.output)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--success-rate", "2"},
		{"--failure-code", "2"},
		{"--log-format", "yaml"},
		{"--duration", "nope"},
	} {
		r := runWobble(t, nil, args...)
		if r.code != 2 {
			t.Errorf("%v: code = %d, want 2\n%s", args, r.code, r.output)
		}
		if !strings.Contains(r.output, "wobble:") {
			t.Errorf("%v: missing diagnostic\n%s", args, r.output)
		}
	}
}

func TestHelpAndVersionExitZero(t *testing.T) {
	h := runWobble(t, nil, "--help")
	if h.code != 0 || !strings.Contains(h.output, "Usage: wobble") {
		t.Errorf("--help: code=%d\n%s", h.code, h.output)
	}
	v := runWobble(t, nil, "--version")
	if v.code != 0 || !strings.HasPrefix(v.output, "wobble ") {
		t.Errorf("--version: code=%d out=%q", v.code, v.output)
	}
}

func TestSIGINT(t *testing.T) {
	cmd, buf := startWobble(t, nil, "--duration", "10s", "--seed", "1")
	time.Sleep(300 * time.Millisecond)
	_ = cmd.Process.Signal(syscall.SIGINT)
	if code := waitCode(t, cmd); code != 130 {
		t.Fatalf("code = %d, want 130\n%s", code, buf.String())
	}
	mustContain(t, buf.String(), "terminal_reason=signal")
}

func TestSIGTERM(t *testing.T) {
	cmd, buf := startWobble(t, nil, "--duration", "10s", "--seed", "1")
	time.Sleep(300 * time.Millisecond)
	_ = cmd.Process.Signal(syscall.SIGTERM)
	if code := waitCode(t, cmd); code != 143 {
		t.Fatalf("code = %d, want 143\n%s", code, buf.String())
	}
}

func TestSignalOverridesSuccessOutcome(t *testing.T) {
	// success-rate 1 => outcome is success, but a signal must still win.
	cmd, buf := startWobble(t, nil, "--duration", "10s", "--success-rate", "1", "--seed", "1")
	time.Sleep(300 * time.Millisecond)
	_ = cmd.Process.Signal(syscall.SIGTERM)
	if code := waitCode(t, cmd); code != 143 {
		t.Fatalf("code = %d, want 143\n%s", code, buf.String())
	}
	out := buf.String()
	mustContain(t, out, "outcome=success")
	mustContain(t, out, "terminal_reason=signal")
}

func TestDoubleSIGINTIsImmediate(t *testing.T) {
	cmd, buf := startWobble(t, []string{"WOBBLE_TEST_WEDGE_SHUTDOWN=5s"}, "--duration", "10s", "--seed", "1")
	time.Sleep(300 * time.Millisecond)
	_ = cmd.Process.Signal(syscall.SIGINT)
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	_ = cmd.Process.Signal(syscall.SIGINT)
	code := waitCode(t, cmd)
	if code != 130 {
		t.Fatalf("code = %d, want 130\n%s", code, buf.String())
	}
	if el := time.Since(start); el > 2*time.Second {
		t.Errorf("second SIGINT took %v to exit; should not wait for wedged shutdown", el)
	}
}

func TestWatchdogForceTerminates(t *testing.T) {
	r := runWobble(t,
		[]string{"WOBBLE_TEST_WEDGE_SHUTDOWN=5s"},
		"--duration", "10s", "--max-duration", "150ms", "--grace", "100ms", "--seed", "1")
	if r.code != 124 {
		t.Fatalf("code = %d, want 124\n%s", r.code, r.output)
	}
	mustContain(t, r.output, "terminal_reason=watchdog")
	mustContain(t, r.output, "exit_code=124")
	if r.elapsed > 3*time.Second {
		t.Errorf("watchdog run took %v; wedge is 5s so watchdog should have fired near 250ms", r.elapsed)
	}
}

func TestReproducibleAcrossRuns(t *testing.T) {
	args := []string{"--duration", "1s..2s", "--cpu", "0.1..0.9", "--success-rate", "0.5", "--seed", "424242", "--verbose"}
	a := firstLineWith(runWobble(t, nil, args...).output, "msg=startup")
	b := firstLineWith(runWobble(t, nil, args...).output, "msg=startup")
	an := stripTime(a)
	bn := stripTime(b)
	if an != bn {
		t.Errorf("same seed produced different startup summaries:\n %s\n %s", an, bn)
	}
}

func firstLineWith(out, sub string) string {
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, sub) {
			return ln
		}
	}
	return ""
}

func stripTime(line string) string {
	// Drop the leading "time=... " field so lines are comparable across runs.
	if i := strings.Index(line, " level="); i >= 0 {
		return line[i:]
	}
	return line
}
