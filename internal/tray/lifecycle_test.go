package tray

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
)

// startApp runs the full OnReady path, including the background goroutines, and
// guarantees the app is stopped when the test ends.
func startApp(t *testing.T, checker monitor.Checker) (*App, *fakeMenu) {
	t.Helper()

	menu := newFakeMenu()
	configuration := config.Default()
	configuration.MaxVisibleUpdates = 3

	app := New(Options{
		Menu:      menu,
		Checker:   checker,
		Config:    configuration,
		Updater:   stubUpdater{},
		Autostart: stubAutostart{status: autostart.Disabled},
		Texts:     localization.MustNew("en"),
	})
	app.OnReady()
	t.Cleanup(app.OnExit)
	return app, menu
}

// Clicking Quit must cancel the app and ask the menu bar to shut down.
func TestQuitClickQuitsTheMenu(t *testing.T) {
	t.Parallel()

	app, menu := startApp(t, countingChecker(nil))
	fakeItem(t, app.quitItem).click()

	waitFor(t, "menu quit", func() bool { return menu.quitCount() == 1 })
}

// Clicking Check now must run a check even though the interval has not elapsed.
func TestRefreshClickTriggersCheck(t *testing.T) {
	t.Parallel()

	var checks atomic.Int32
	app, _ := startApp(t, countingChecker(&checks))

	// The startup check happens on its own.
	waitFor(t, "startup check", func() bool { return checks.Load() >= 1 })

	fakeItem(t, app.refreshItem).click()
	waitFor(t, "triggered check", func() bool { return checks.Load() >= 2 })
}

// Finishing an interactive Homebrew upgrade must trigger one fresh check so
// the menu reflects the packages that are still outdated.
func TestUpgradeCompletionTriggersCheck(t *testing.T) {
	t.Parallel()

	var checks atomic.Int32
	completed := make(chan struct{}, 1)
	menu := newFakeMenu()
	configuration := config.Default()
	configuration.MaxVisibleUpdates = 3
	app := New(Options{
		Menu:      menu,
		Checker:   countingChecker(&checks),
		Config:    configuration,
		Updater:   stubUpdater{completed: completed},
		Autostart: stubAutostart{status: autostart.Disabled},
		Texts:     localization.MustNew("en"),
	})
	app.OnReady()
	t.Cleanup(app.OnExit)

	waitFor(t, "startup check", func() bool { return checks.Load() >= 1 })
	completed <- struct{}{}
	waitFor(t, "post-upgrade check", func() bool { return checks.Load() >= 2 })
}

// OnExit must return once every background task has stopped. A goroutine that
// is not registered with the wait group would make this hang, which is what the
// previous hard-coded counter risked on every added task.
func TestOnExitStopsAllBackgroundTasks(t *testing.T) {
	t.Parallel()

	menu := newFakeMenu()
	app := New(Options{
		Menu:      menu,
		Checker:   countingChecker(nil),
		Config:    config.Default(),
		Autostart: stubAutostart{status: autostart.Disabled},
		Texts:     localization.MustNew("en"),
	})
	app.OnReady()

	finished := make(chan struct{})
	go func() {
		app.OnExit()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("OnExit did not return; a background task is not tracked")
	}
}

func countingChecker(counter *atomic.Int32) monitor.Checker {
	return monitor.CheckerFunc(func(context.Context) (brew.Result, error) {
		if counter != nil {
			counter.Add(1)
		}
		return brew.Result{CheckedAt: time.Now()}, nil
	})
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
