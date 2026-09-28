package tray

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
)

// newTestApp builds an application against a fake menu and runs OnReady so that
// all menu items exist, without starting any background goroutine.
func newTestApp(t *testing.T, maxVisible int) (*App, *fakeMenu) {
	t.Helper()

	menu := newFakeMenu()
	configuration := config.Default()
	configuration.MaxVisibleUpdates = maxVisible

	app := New(Options{
		Menu:      menu,
		Checker:   monitor.CheckerFunc(func(context.Context) (brew.Result, error) { return brew.Result{}, nil }),
		Config:    configuration,
		Updater:   stubUpdater{},
		Autostart: stubAutostart{status: autostart.Disabled},
		Texts:     localization.MustNew("en"),
	})
	app.buildMenu()
	return app, menu
}

func TestRenderResultShowsPackagesAndCount(t *testing.T) {
	t.Parallel()

	app, menu := newTestApp(t, 10)
	app.renderResult(brew.Result{
		Packages: []brew.Package{
			makePackage("go", "1.26.5", "1.26.6"),
			makePackage("node", "24.0.0", "25.0.0"),
		},
		CheckedAt: time.Date(2026, time.September, 27, 9, 0, 0, 0, time.UTC),
	})

	if menu.currentTitle() != "2" {
		t.Errorf("menu bar title = %q, want %q", menu.currentTitle(), "2")
	}
	if !visible(t, app.updateItems[0]) || !visible(t, app.updateItems[1]) {
		t.Error("the two package rows should be visible")
	}
	if visible(t, app.updateItems[2]) {
		t.Error("the third row should stay hidden with only two packages")
	}
	if !visible(t, app.updateAllItem) || !enabled(t, app.updateAllItem) {
		t.Error("the update-all row should be visible and enabled")
	}
	if visible(t, app.overflow) {
		t.Error("the overflow row should stay hidden when everything fits")
	}
}

func TestRenderResultEmptyClearsMenuBarTitle(t *testing.T) {
	t.Parallel()

	app, menu := newTestApp(t, 10)

	// Render a populated result first so the empty render has to clean up.
	app.renderResult(brew.Result{Packages: []brew.Package{makePackage("go", "1", "2")}})
	app.renderResult(brew.Result{})

	if menu.currentTitle() != "" {
		t.Errorf("menu bar title = %q, want it empty when up to date", menu.currentTitle())
	}
	if visible(t, app.updateItems[0]) {
		t.Error("package rows should be hidden when nothing is outdated")
	}
	if visible(t, app.updateAllItem) {
		t.Error("the update-all row should be hidden when nothing is outdated")
	}
	if titleOf(t, app.statusItem) != localization.MustNew("en").UpToDate() {
		t.Errorf("status = %q, want the up-to-date text", titleOf(t, app.statusItem))
	}
}

// The configured row limit must be respected and the remainder summarized.
func TestRenderResultOverflowRespectsConfiguredLimit(t *testing.T) {
	t.Parallel()

	const limit = 3
	app, _ := newTestApp(t, limit)

	packages := make([]brew.Package, 0, 7)
	for index := range 7 {
		packages = append(packages, makePackage(fmt.Sprintf("pkg%d", index), "1", "2"))
	}
	app.renderResult(brew.Result{Packages: packages})

	if len(app.updateItems) != limit {
		t.Fatalf("created %d rows, want %d", len(app.updateItems), limit)
	}
	for index, item := range app.updateItems {
		if !visible(t, item) {
			t.Errorf("row %d should be visible", index)
		}
	}
	if !visible(t, app.overflow) {
		t.Fatal("the overflow row should be visible when packages exceed the limit")
	}
	// 7 packages minus 3 visible rows.
	want := localization.MustNew("en").MoreUpdates(4)
	if titleOf(t, app.overflow) != want {
		t.Errorf("overflow = %q, want %q", titleOf(t, app.overflow), want)
	}
}

// Without an updater the rows must stay visible but inert, so that a missing
// Homebrew cannot produce a click that goes nowhere.
func TestRenderResultDisablesActionsWithoutUpdater(t *testing.T) {
	t.Parallel()

	menu := newFakeMenu()
	app := New(Options{
		Menu:      menu,
		Checker:   monitor.CheckerFunc(func(context.Context) (brew.Result, error) { return brew.Result{}, nil }),
		Config:    config.Default(),
		Updater:   nil,
		Autostart: stubAutostart{status: autostart.Disabled},
		Texts:     localization.MustNew("en"),
	})
	app.buildMenu()

	app.renderResult(brew.Result{Packages: []brew.Package{makePackage("go", "1", "2")}})

	if enabled(t, app.updateItems[0]) {
		t.Error("package row should be disabled without an updater")
	}
	if enabled(t, app.updateAllItem) {
		t.Error("update-all row should be disabled without an updater")
	}
	if !visible(t, app.updateItems[0]) {
		t.Error("package row should still be visible so the update is discoverable")
	}
}

func TestRenderResultLocalizesMetadataRefreshWarning(t *testing.T) {
	t.Parallel()

	app, _ := newTestApp(t, 10)
	app.renderResult(brew.Result{
		CheckedAt: time.Date(2026, time.September, 27, 9, 0, 0, 0, time.UTC),
		Warning: &brew.Warning{
			Kind:  brew.WarningMetadataRefreshFailed,
			Cause: errors.New("offline"),
		},
	})

	if got := tooltipOf(t, app.checkedItem); got != localization.MustNew("en").MetadataRefreshWarningTooltip() {
		t.Errorf("warning tooltip = %q, want localized metadata warning", got)
	}
}

func TestRenderErrorHidesUpdatesAndMarksMenuBar(t *testing.T) {
	t.Parallel()

	app, menu := newTestApp(t, 10)
	app.renderResult(brew.Result{Packages: []brew.Package{makePackage("go", "1", "2")}})

	app.render(monitor.State{Err: fmt.Errorf("%w: exit status 1", brew.ErrQueryFailed)})

	if menu.currentTitle() != "!" {
		t.Errorf("menu bar title = %q, want %q", menu.currentTitle(), "!")
	}
	if visible(t, app.updateItems[0]) {
		t.Error("package rows should be hidden after a failed check")
	}
	if visible(t, app.updateAllItem) {
		t.Error("the update-all row should be hidden after a failed check")
	}
}

func TestRenderCheckingDisablesRefresh(t *testing.T) {
	t.Parallel()

	app, menu := newTestApp(t, 10)
	app.render(monitor.State{Checking: true})

	if menu.currentTitle() != "…" {
		t.Errorf("menu bar title = %q, want the checking indicator", menu.currentTitle())
	}
	if enabled(t, app.refreshItem) {
		t.Error("the refresh row should be disabled while a check runs")
	}

	app.render(monitor.State{Result: &brew.Result{}})
	if !enabled(t, app.refreshItem) {
		t.Error("the refresh row should be enabled again after a check")
	}
}

// Every classified Homebrew failure must produce a localized row rather than
// the raw English error text.
func TestCheckErrorMessageIsLocalized(t *testing.T) {
	t.Parallel()

	german := localization.MustNew("de")
	english := localization.MustNew("en")

	tests := []struct {
		name string
		err  error
		want func(*localization.Strings) string
	}{
		{
			name: "not found",
			err:  fmt.Errorf("%w: not in PATH", brew.ErrNotFound),
			want: (*localization.Strings).HomebrewNotFound,
		},
		{
			name: "timeout",
			err:  fmt.Errorf("%w: %w", brew.ErrTimeout, context.DeadlineExceeded),
			want: (*localization.Strings).HomebrewTimeout,
		},
		{
			name: "invalid output",
			err:  fmt.Errorf("%w: bad json", brew.ErrInvalidOutput),
			want: (*localization.Strings).HomebrewUnexpectedOutput,
		},
		{
			name: "query failed",
			err:  fmt.Errorf("%w: exit status 1", brew.ErrQueryFailed),
			want: (*localization.Strings).HomebrewQueryFailed,
		},
		{
			name: "unclassified",
			err:  errors.New("something else entirely"),
			want: (*localization.Strings).UnexpectedError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := checkErrorMessage(german, test.err)
			if want := test.want(german); got != want {
				t.Errorf("German message = %q, want %q", got, want)
			}
			// The same failure in English must differ, proving the text really
			// comes from the catalog rather than from the error itself.
			if got == checkErrorMessage(english, test.err) && test.name != "unclassified" {
				t.Errorf("German and English message are identical: %q", got)
			}
		})
	}
}

// A very long brew failure must not be able to blow up a tooltip, and must
// never be cut inside a multi-byte character.
func TestShortErrorTruncatesOnRuneBoundary(t *testing.T) {
	t.Parallel()

	long := ""
	for range 400 {
		long += "ü"
	}
	got := shortError(errors.New(long))

	if !utf8Valid(got) {
		t.Fatalf("shortError produced invalid UTF-8: %q", got)
	}
	if len([]rune(got)) != 201 { // 200 runes plus the ellipsis
		t.Fatalf("shortError returned %d runes, want 201", len([]rune(got)))
	}
}

func TestShortErrorKeepsShortMessages(t *testing.T) {
	t.Parallel()

	if got := shortError(errors.New("short")); got != "short" {
		t.Fatalf("shortError() = %q, want %q", got, "short")
	}
}

func makePackage(name, installed, current string) brew.Package {
	return brew.Package{
		Name:              name,
		Kind:              brew.Formula,
		InstalledVersions: []string{installed},
		CurrentVersion:    current,
	}
}

func utf8Valid(value string) bool {
	for _, r := range value {
		if r == '�' {
			return false
		}
	}
	return true
}

type stubUpdater struct{ err error }

func (updater stubUpdater) UpgradePackage(brew.Package) error { return updater.err }
func (updater stubUpdater) UpgradeAll() error                 { return updater.err }

type stubAutostart struct {
	status autostart.Status
	err    error
}

func (controller stubAutostart) Status() (autostart.Status, error) {
	return controller.status, controller.err
}

func (controller stubAutostart) SetEnabled(bool) (autostart.Status, error) {
	return controller.status, controller.err
}

func (controller stubAutostart) OpenSettings() error { return controller.err }
