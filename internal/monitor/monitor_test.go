package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
)

func TestMonitorChecksImmediately(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := brew.Result{CheckedAt: time.Now()}
	checker := CheckerFunc(func(context.Context) (brew.Result, error) {
		return result, nil
	})

	var mutex sync.Mutex
	var states []State
	finished := make(chan struct{})
	monitor := New(checker, time.Hour, func(state State) {
		mutex.Lock()
		states = append(states, state)
		mutex.Unlock()
		if state.Result != nil {
			select {
			case <-finished:
			default:
				close(finished)
			}
		}
	})

	go monitor.Run(ctx)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("monitor did not check immediately")
	}
	cancel()

	mutex.Lock()
	defer mutex.Unlock()
	if len(states) < 2 || !states[0].Checking || states[1].Result == nil {
		t.Fatalf("states = %#v, want checking state followed by result", states)
	}
}
