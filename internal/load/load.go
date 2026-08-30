// Package load generates a roughly constant amount of CPU load for the life of a
// wobble run. Load is expressed in "busy cores"; see openspec/specs/cpu-load/spec.md.
package load

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// controlInterval is the busy/idle cycle length. A worker re-checks its context
// far more often than this, so cancellation is honoured well within it.
const controlInterval = 100 * time.Millisecond

// sink keeps the busy-loop arithmetic observable so the compiler cannot elide it.
var sink atomic.Uint64

// Duties returns the per-worker duty cycles for a resolved CPU target.
//
//   - override <= 0: derive ceil(target) workers; worker i gets clamp(target-i, 0, 1)
//     so the first floor(target) workers run flat-out and the last carries the
//     fractional remainder.
//   - override > 0: exactly override workers, each with clamp(target/override, 0, 1).
//
// Returns nil when no workers should run.
func Duties(target float64, override int) []float64 {
	if override > 0 {
		d := clamp(target/float64(override), 0, 1)
		out := make([]float64, override)
		for i := range out {
			out[i] = d
		}
		return out
	}
	if target <= 0 {
		return nil
	}
	n := int(math.Ceil(target))
	if n < 1 {
		n = 1
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = clamp(target-float64(i), 0, 1)
	}
	return out
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Pool is a set of running load workers.
type Pool struct {
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	stopOnce sync.Once

	ran bool

	cpu0, cpu1         float64
	haveCPU0, haveCPU1 bool
	t0, t1             time.Time
}

// Start launches one worker per entry in duties, bound to ctx. If duties is
// empty no workers run and the returned Pool is inert. Stop must still be called.
func Start(ctx context.Context, duties []float64) *Pool {
	cctx, cancel := context.WithCancel(ctx)
	p := &Pool{cancel: cancel}
	if len(duties) == 0 {
		return p
	}

	p.ran = true
	p.cpu0, p.haveCPU0 = processCPUSeconds()
	p.t0 = time.Now()

	for _, d := range duties {
		p.wg.Add(1)
		go func(duty float64) {
			defer p.wg.Done()
			worker(cctx, duty)
		}(d)
	}
	return p
}

// Stop signals every worker to stop, records the closing CPU sample, and waits
// for the workers to exit. Safe to call more than once.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		if p.ran {
			p.cpu1, p.haveCPU1 = processCPUSeconds()
			p.t1 = time.Now()
		}
		p.cancel()
		p.wg.Wait()
	})
}

// Ran reports whether any worker was started.
func (p *Pool) Ran() bool { return p.ran }

// ObservedCores returns the average busy-core count over the pool's active
// window. Valid only after Stop, and only when CPU accounting is available on
// this platform.
func (p *Pool) ObservedCores() (float64, bool) {
	if !p.ran || !p.haveCPU0 || !p.haveCPU1 {
		return 0, false
	}
	wall := p.t1.Sub(p.t0).Seconds()
	if wall <= 0 {
		return 0, false
	}
	return (p.cpu1 - p.cpu0) / wall, true
}

func worker(ctx context.Context, duty float64) {
	if duty <= 0 {
		<-ctx.Done()
		return
	}
	busy := time.Duration(float64(controlInterval) * duty)
	idle := controlInterval - busy

	for {
		if ctx.Err() != nil {
			return
		}

		start := time.Now()
		var acc float64
		for time.Since(start) < busy {
			for i := 0; i < 4096; i++ {
				acc += math.Sqrt(float64(i) + 1)
				if acc > 1e9 {
					acc -= 1e9
				}
			}
			if ctx.Err() != nil {
				sink.Add(uint64(acc))
				return
			}
		}
		sink.Add(uint64(acc))

		if idle > 0 {
			t := time.NewTimer(idle)
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
			}
		}
	}
}
