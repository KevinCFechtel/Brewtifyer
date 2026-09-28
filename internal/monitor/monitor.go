// Package monitor schedules Homebrew checks and guarantees that only one
// check runs at a time. Scheduling is wall-clock based so that the interval
// holds across macOS sleep.
package monitor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
)

// pollInterval is how often the monitor compares the wall clock against the
// configured interval. It is deliberately much shorter than the interval
// itself; see Run for why a plain long ticker is not sufficient.
const pollInterval = time.Minute

type Checker interface {
	Check(ctx context.Context) (brew.Result, error)
}

type CheckerFunc func(context.Context) (brew.Result, error)

func (function CheckerFunc) Check(ctx context.Context) (brew.Result, error) {
	return function(ctx)
}

type State struct {
	Checking bool
	Result   *brew.Result
	Err      error
}

// Monitor serializes automatic and manually requested checks.
type Monitor struct {
	checker      Checker
	interval     time.Duration
	pollInterval time.Duration
	now          func() time.Time
	onState      func(State)
	trigger      chan struct{}
	running      atomic.Bool

	lastMutex     sync.Mutex
	lastCheckedAt time.Time
}

func New(checker Checker, interval time.Duration, onState func(State)) *Monitor {
	poll := pollInterval
	if interval < poll {
		poll = interval
	}
	return &Monitor{
		checker:      checker,
		interval:     interval,
		pollInterval: poll,
		now:          time.Now,
		onState:      onState,
		trigger:      make(chan struct{}, 1),
	}
}

// Run checks immediately and then keeps checking roughly every interval.
//
// The schedule is driven by the wall clock rather than by a single long ticker.
// On macOS the monotonic clock that timers are based on does not advance while
// the system sleeps, so a six-hour ticker on a laptop that is closed overnight
// fires hours late and the menu keeps presenting stale data as current.
// Waking up every pollInterval and comparing real timestamps makes the interval
// hold across sleep, at the cost of a cheap comparison per minute.
func (monitor *Monitor) Run(ctx context.Context) {
	monitor.check(ctx)

	ticker := time.NewTicker(monitor.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if monitor.due() {
				monitor.check(ctx)
			}
		case <-monitor.trigger:
			monitor.check(ctx)
		}
	}
}

// Trigger requests a check outside the schedule, for example after the user
// picked "Check now".
//
// A request that arrives while a check is running is queued rather than
// dropped: the running check started before the user asked, so its data can
// already be stale by the time it lands. The buffer of one collapses repeated
// clicks into a single pending check, and check itself keeps the checks
// serialized, so at most one extra check follows.
func (monitor *Monitor) Trigger() {
	select {
	case monitor.trigger <- struct{}{}:
	default:
	}
}

// due reports whether at least interval of wall-clock time has passed since the
// last completed check.
func (monitor *Monitor) due() bool {
	monitor.lastMutex.Lock()
	defer monitor.lastMutex.Unlock()

	if monitor.lastCheckedAt.IsZero() {
		return true
	}
	return wallClock(monitor.now()).Sub(wallClock(monitor.lastCheckedAt)) >= monitor.interval
}

// wallClock strips Go's monotonic clock reading. Scheduling is intentionally
// based on civil time because mach_absolute_time does not advance while macOS
// sleeps; retaining the monotonic component would make a closed laptop appear
// to have slept for zero time.
func wallClock(value time.Time) time.Time {
	return value.Round(0)
}

func (monitor *Monitor) check(ctx context.Context) {
	if !monitor.running.CompareAndSwap(false, true) {
		return
	}
	defer monitor.running.Store(false)

	monitor.onState(State{Checking: true})
	result, err := monitor.checker.Check(ctx)

	// The attempt counts as the last check even when it failed, so that a
	// permanently broken Homebrew is not retried every pollInterval.
	monitor.lastMutex.Lock()
	monitor.lastCheckedAt = wallClock(monitor.now())
	monitor.lastMutex.Unlock()

	if err != nil {
		monitor.onState(State{Err: err})
		return
	}
	monitor.onState(State{Result: &result})
}
