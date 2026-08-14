package monitor

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
)

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
	checker  Checker
	interval time.Duration
	onState  func(State)
	trigger  chan struct{}
	running  atomic.Bool
}

func New(checker Checker, interval time.Duration, onState func(State)) *Monitor {
	return &Monitor{
		checker:  checker,
		interval: interval,
		onState:  onState,
		trigger:  make(chan struct{}, 1),
	}
}

func (monitor *Monitor) Run(ctx context.Context) {
	monitor.check(ctx)

	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitor.check(ctx)
		case <-monitor.trigger:
			monitor.check(ctx)
		}
	}
}

func (monitor *Monitor) Trigger() {
	if monitor.running.Load() {
		return
	}
	select {
	case monitor.trigger <- struct{}{}:
	default:
	}
}

func (monitor *Monitor) check(ctx context.Context) {
	if !monitor.running.CompareAndSwap(false, true) {
		return
	}
	defer monitor.running.Store(false)

	monitor.onState(State{Checking: true})
	result, err := monitor.checker.Check(ctx)
	if err != nil {
		monitor.onState(State{Err: err})
		return
	}
	monitor.onState(State{Result: &result})
}
