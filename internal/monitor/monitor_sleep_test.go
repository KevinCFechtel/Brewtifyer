package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
)

// The scheduling decision must come from the wall clock. A monotonic clock
// stops during system sleep, which is what made a single long ticker fire late
// after the lid was closed.
func TestDueUsesWallClock(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, time.September, 27, 8, 0, 0, 0, time.UTC)
	monitor := New(noopChecker(), 6*time.Hour, func(State) {})
	monitor.now = func() time.Time { return current }

	if !monitor.due() {
		t.Fatal("due() = false before the first check, want true")
	}

	monitor.lastCheckedAt = current
	if monitor.due() {
		t.Fatal("due() = true immediately after a check, want false")
	}

	// Not yet enough wall-clock time.
	current = current.Add(5*time.Hour + 59*time.Minute)
	if monitor.due() {
		t.Fatal("due() = true before the interval elapsed, want false")
	}

	// Simulates the machine having been asleep past the interval: only the wall
	// clock moved, no timer fired in between.
	current = current.Add(2 * time.Minute)
	if !monitor.due() {
		t.Fatal("due() = false after the interval elapsed, want true")
	}
}

// A very short interval must not produce a poll interval longer than itself,
// otherwise the configured interval would be silently ignored.
func TestPollIntervalNeverExceedsInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		interval time.Duration
		wantPoll time.Duration
	}{
		{interval: 6 * time.Hour, wantPoll: pollInterval},
		{interval: 30 * time.Second, wantPoll: 30 * time.Second},
		{interval: pollInterval, wantPoll: pollInterval},
	}

	for _, test := range tests {
		monitor := New(noopChecker(), test.interval, func(State) {})
		if monitor.pollInterval != test.wantPoll {
			t.Errorf("New(%v).pollInterval = %v, want %v",
				test.interval, monitor.pollInterval, test.wantPoll)
		}
	}
}

// A failing check must still update the timestamp, so that a broken Homebrew is
// retried on the normal interval instead of on every poll tick.
func TestFailedCheckStillMarksTime(t *testing.T) {
	t.Parallel()

	current := time.Date(2026, time.September, 27, 8, 0, 0, 0, time.UTC)
	var calls int
	var mutex sync.Mutex
	monitor := New(CheckerFunc(func(context.Context) (brew.Result, error) {
		mutex.Lock()
		calls++
		mutex.Unlock()
		return brew.Result{}, context.DeadlineExceeded
	}), time.Hour, func(State) {})
	monitor.now = func() time.Time { return current }

	monitor.check(context.Background())

	mutex.Lock()
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	mutex.Unlock()

	if monitor.due() {
		t.Fatal("due() = true right after a failed check, want false")
	}
}

// A manual trigger must bypass the interval entirely.
func TestTriggerChecksRegardlessOfInterval(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan struct{}, 4)
	monitor := New(noopChecker(), time.Hour, func(state State) {
		if state.Result != nil {
			results <- struct{}{}
		}
	})

	go monitor.Run(ctx)

	// The immediate startup check.
	waitForResult(t, results, "startup check")

	monitor.Trigger()
	waitForResult(t, results, "triggered check")
}

func waitForResult(t *testing.T, results <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-results:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s did not complete", what)
	}
}

func noopChecker() Checker {
	return CheckerFunc(func(context.Context) (brew.Result, error) {
		return brew.Result{}, nil
	})
}
