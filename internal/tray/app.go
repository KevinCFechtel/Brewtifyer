// Package tray owns the menu bar interface. The menu bar library is reached
// only through the Menu and MenuItem interfaces in menu.go, which keeps the
// render logic testable and the library replaceable.
package tray

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
)

const autostartRefreshInterval = 5 * time.Second

type Updater interface {
	UpgradePackage(brew.Package) error
	UpgradeAll() error
}

// Options collects everything the menu bar application depends on. It is a
// struct rather than a parameter list so that a new dependency does not break
// every caller.
type Options struct {
	// Menu is the menu bar backend. Required.
	Menu Menu
	// Checker performs one Homebrew check. Required.
	Checker monitor.Checker
	// Config supplies the check interval and the menu limits.
	Config config.Config
	// ResultHandler is called after every successful check. Optional.
	ResultHandler func(brew.Result)
	// Updater opens interactive upgrades. Nil disables the upgrade actions.
	Updater Updater
	// Autostart manages the login item. Nil reports the feature unsupported.
	Autostart autostart.Controller
	// Texts supplies all user-facing strings. Required.
	Texts *localization.Strings
}

type App struct {
	menu          Menu
	checker       monitor.Checker
	configuration config.Config
	resultHandler func(brew.Result)
	updater       Updater
	autostart     autostart.Controller
	texts         *localization.Strings

	ctx             context.Context
	cancel          context.CancelFunc
	monitor         *monitor.Monitor
	wait            sync.WaitGroup
	packagesMutex   sync.RWMutex
	currentPackages []brew.Package

	statusItem            MenuItem
	checkedItem           MenuItem
	updateItems           []MenuItem
	overflow              MenuItem
	updateAllItem         MenuItem
	refreshItem           MenuItem
	autostartItem         MenuItem
	autostartSettingsItem MenuItem
	quitItem              MenuItem
}

func New(options Options) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		menu:          options.Menu,
		checker:       options.Checker,
		configuration: options.Config,
		resultHandler: options.ResultHandler,
		updater:       options.Updater,
		autostart:     options.Autostart,
		texts:         options.Texts,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// OnReady is the systray entry point: it builds the menu and starts the
// background tasks.
func (app *App) OnReady() {
	app.buildMenu()
	app.monitor = monitor.New(app.checker, app.configuration.CheckInterval, app.render)
	app.refreshAutostart()
	app.startBackgroundTasks()
}

// buildMenu creates every menu item in its initial state. It is separate from
// OnReady so that the render logic can be exercised against a fake menu without
// starting any goroutine.
func (app *App) buildMenu() {
	app.menu.SetTemplateIcon(iconPNG())
	app.menu.SetTooltip(app.texts.TrayTooltip())
	app.menu.SetRemovalAllowed(false)

	app.statusItem = app.menu.AddItem(app.texts.Checking(), app.texts.CurrentStatusTooltip())
	app.statusItem.Disable()
	app.checkedItem = app.menu.AddItem(app.texts.NotChecked(), app.texts.LastCheckTooltip())
	app.checkedItem.Disable()
	app.menu.AddSeparator()

	for range app.configuration.MaxVisibleUpdates {
		item := app.menu.AddItem("", app.texts.UpgradePackageMenuTooltip())
		item.Disable()
		item.Hide()
		app.updateItems = append(app.updateItems, item)
	}
	app.overflow = app.menu.AddItem("", "")
	app.overflow.Disable()
	app.overflow.Hide()
	app.updateAllItem = app.menu.AddItem(app.texts.UpgradeAll(), app.texts.UpgradeAllTooltip())
	app.updateAllItem.Disable()
	app.updateAllItem.Hide()

	app.menu.AddSeparator()
	app.refreshItem = app.menu.AddItem(app.texts.Refresh(), app.texts.RefreshTooltip())
	app.autostartItem = app.menu.AddCheckbox(
		app.texts.AutostartTitle(), app.texts.AutostartEnableTooltip(), false)
	app.autostartSettingsItem = app.menu.AddItem(
		app.texts.OpenLoginItems(),
		app.texts.OpenLoginItemsTooltip(),
	)
	app.autostartSettingsItem.Hide()
	app.menu.AddSeparator()
	app.quitItem = app.menu.AddItem(app.texts.Quit(), app.texts.QuitTooltip())
}

func (app *App) OnExit() {
	app.cancel()
	app.wait.Wait()
}

// goroutine starts a tracked background task. Registering the task with the
// wait group here, rather than counting the goroutines up front, keeps OnExit
// correct when a task is added or removed.
func (app *App) goroutine(task func()) {
	app.wait.Add(1)
	go func() {
		defer app.wait.Done()
		task()
	}()
}

func (app *App) startBackgroundTasks() {
	app.goroutine(func() { app.monitor.Run(app.ctx) })

	app.goroutine(func() {
		app.consumeClicks(app.refreshItem, func() { app.monitor.Trigger() })
	})

	app.goroutine(func() {
		select {
		case <-app.ctx.Done():
		case <-app.quitItem.Clicked():
			app.cancel()
			app.menu.Quit()
		}
	})

	for index, item := range app.updateItems {
		app.goroutine(func() {
			app.consumeClicks(item, func() { app.upgradePackage(index) })
		})
	}

	app.goroutine(func() {
		app.consumeClicks(app.updateAllItem, app.upgradeAll)
	})

	app.goroutine(app.watchAutostart)
}

// consumeClicks runs handle for every activation of item until the app stops.
func (app *App) consumeClicks(item MenuItem, handle func()) {
	clicked := item.Clicked()
	for {
		select {
		case <-app.ctx.Done():
			return
		case _, open := <-clicked:
			if !open {
				return
			}
			handle()
		}
	}
}

func (app *App) watchAutostart() {
	ticker := time.NewTicker(autostartRefreshInterval)
	defer ticker.Stop()

	autostartClicked := app.autostartItem.Clicked()
	settingsClicked := app.autostartSettingsItem.Clicked()
	for {
		select {
		case <-app.ctx.Done():
			return
		case _, open := <-autostartClicked:
			if !open {
				return
			}
			app.toggleAutostart()
		case _, open := <-settingsClicked:
			if !open {
				return
			}
			app.openAutostartSettings()
		case <-ticker.C:
			app.refreshAutostart()
		}
	}
}

func (app *App) render(state monitor.State) {
	if state.Checking {
		app.menu.SetTitle("…")
		app.statusItem.SetTitle(app.texts.Checking())
		app.refreshItem.Disable()
		return
	}

	app.refreshItem.Enable()
	if state.Err != nil {
		// The menu shows a localized explanation; the log keeps the technical
		// cause, which is usually the only thing a bug report can act on.
		log.Printf("Homebrew check failed: %v", state.Err)
		app.menu.SetTitle("!")
		app.statusItem.SetTitle(app.texts.CheckFailed())
		app.checkedItem.SetTitle(checkErrorMessage(app.texts, state.Err))
		app.checkedItem.SetTooltip(checkErrorTooltip(app.texts, state.Err))
		app.hideUpdates()
		return
	}
	if state.Result == nil {
		return
	}

	app.renderResult(*state.Result)
	if app.resultHandler != nil {
		app.resultHandler(*state.Result)
	}
}

func (app *App) renderResult(result brew.Result) {
	count := len(result.Packages)
	app.packagesMutex.Lock()
	app.currentPackages = append(app.currentPackages[:0], result.Packages...)
	app.packagesMutex.Unlock()

	if count == 0 {
		app.menu.SetTitle("")
		app.statusItem.SetTitle(app.texts.UpToDate())
	} else {
		app.menu.SetTitle(strconv.Itoa(count))
		app.statusItem.SetTitle(app.texts.UpdatesAvailable(count))
	}

	checkedTitle := app.texts.LastChecked(result.CheckedAt, result.Warning != "")
	if result.Warning != "" {
		app.checkedItem.SetTooltip(result.Warning)
	} else {
		app.checkedItem.SetTooltip(app.texts.LastSuccessfulCheckTooltip())
	}
	app.checkedItem.SetTitle(checkedTitle)

	for index, item := range app.updateItems {
		if index >= count {
			item.Disable()
			item.Hide()
			continue
		}
		item.SetTitle(packageTitle(app.texts, result.Packages[index]))
		item.SetTooltip(packageUpdateTooltip(app.texts, result.Packages[index]))
		if app.updater != nil {
			item.Enable()
		} else {
			item.Disable()
		}
		item.Show()
	}

	remaining := count - len(app.updateItems)
	if remaining > 0 {
		app.overflow.SetTitle(app.texts.MoreUpdates(remaining))
		app.overflow.Show()
	} else {
		app.overflow.Hide()
	}

	if count > 0 {
		if app.updater != nil {
			app.updateAllItem.Enable()
		} else {
			app.updateAllItem.Disable()
		}
		app.updateAllItem.Show()
	} else {
		app.updateAllItem.Disable()
		app.updateAllItem.Hide()
	}
}

func (app *App) hideUpdates() {
	app.packagesMutex.Lock()
	app.currentPackages = nil
	app.packagesMutex.Unlock()

	for _, item := range app.updateItems {
		item.Disable()
		item.Hide()
	}
	app.overflow.Hide()
	app.updateAllItem.Disable()
	app.updateAllItem.Hide()
}

func (app *App) upgradePackage(index int) {
	if app.updater == nil {
		return
	}

	app.packagesMutex.RLock()
	if index < 0 || index >= len(app.currentPackages) {
		app.packagesMutex.RUnlock()
		return
	}
	currentPackage := app.currentPackages[index]
	app.packagesMutex.RUnlock()

	if err := app.updater.UpgradePackage(currentPackage); err != nil {
		app.reportUpgradeError(err)
		return
	}
	app.statusItem.SetTitle(app.texts.UpgradePackageRunning())
}

func (app *App) upgradeAll() {
	if app.updater == nil {
		return
	}
	if err := app.updater.UpgradeAll(); err != nil {
		app.reportUpgradeError(err)
		return
	}
	app.statusItem.SetTitle(app.texts.UpgradeAllRunning())
}

func (app *App) reportUpgradeError(err error) {
	log.Printf("Homebrew upgrade could not be started: %v", err)
	app.statusItem.SetTitle(app.texts.UpgradeLaunchFailed())
	app.statusItem.SetTooltip(err.Error())
}

type autostartMenuState struct {
	title        string
	tooltip      string
	checked      bool
	enabled      bool
	showSettings bool
}

func (app *App) refreshAutostart() {
	if app.autostart == nil {
		app.applyAutostartMenuState(autostartMenuStateFor(app.texts, autostart.Unsupported))
		return
	}
	status, err := app.autostart.Status()
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(app.texts, status))
}

func (app *App) toggleAutostart() {
	if app.autostart == nil {
		return
	}
	status, err := app.autostart.Status()
	if err != nil {
		app.reportAutostartError(err)
		return
	}

	if status == autostart.RequiresApproval {
		app.openAutostartSettings()
		return
	}
	desiredEnabled, canToggle := autostartToggle(status)
	if !canToggle {
		app.applyAutostartMenuState(autostartMenuStateFor(app.texts, status))
		return
	}

	resultingStatus, err := app.autostart.SetEnabled(desiredEnabled)
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(app.texts, resultingStatus))
	if resultingStatus == autostart.RequiresApproval {
		app.openAutostartSettings()
	}
}

func (app *App) openAutostartSettings() {
	if app.autostart == nil {
		return
	}
	if err := app.autostart.OpenSettings(); err != nil {
		app.reportAutostartError(err)
	}
}

func (app *App) reportAutostartError(err error) {
	log.Printf("launch at login could not be managed: %v", err)
	app.autostartItem.SetTitle(app.texts.AutostartManageFailed())
	app.autostartItem.SetTooltip(err.Error())
	app.autostartItem.Disable()
}

func (app *App) applyAutostartMenuState(menuState autostartMenuState) {
	app.autostartItem.SetTitle(menuState.title)
	app.autostartItem.SetTooltip(menuState.tooltip)
	if menuState.checked {
		app.autostartItem.Check()
	} else {
		app.autostartItem.Uncheck()
	}
	if menuState.enabled {
		app.autostartItem.Enable()
	} else {
		app.autostartItem.Disable()
	}
	if menuState.showSettings {
		app.autostartSettingsItem.Show()
	} else {
		app.autostartSettingsItem.Hide()
	}
}

func autostartMenuStateFor(texts *localization.Strings, status autostart.Status) autostartMenuState {
	switch status {
	case autostart.Disabled:
		return autostartMenuState{
			title:   texts.AutostartTitle(),
			tooltip: texts.AutostartEnableTooltip(),
			enabled: true,
		}
	case autostart.Enabled:
		return autostartMenuState{
			title:   texts.AutostartTitle(),
			tooltip: texts.AutostartDisableTooltip(),
			checked: true,
			enabled: true,
		}
	case autostart.RequiresApproval:
		return autostartMenuState{
			title:        texts.AutostartApprovalTitle(),
			tooltip:      texts.AutostartApprovalTooltip(),
			enabled:      true,
			showSettings: true,
		}
	case autostart.NotFound:
		return autostartMenuState{
			title:   texts.AutostartTitle(),
			tooltip: texts.AutostartRegisterTooltip(),
			enabled: true,
		}
	default:
		return autostartMenuState{
			title:   texts.AutostartUnsupportedTitle(),
			tooltip: texts.AutostartUnsupportedTooltip(),
		}
	}
}

func autostartToggle(status autostart.Status) (enabled bool, canToggle bool) {
	switch status {
	case autostart.Disabled, autostart.NotFound:
		return true, true
	case autostart.Enabled:
		return false, true
	default:
		return false, false
	}
}

// checkErrorMessage turns a check failure into a localized menu row. Matching
// on the sentinel errors of the brew package keeps the user interface free of
// English error text without making brew depend on localization.
func checkErrorMessage(texts *localization.Strings, err error) string {
	switch {
	case errors.Is(err, brew.ErrNotFound):
		return texts.HomebrewNotFound()
	case errors.Is(err, brew.ErrTimeout):
		return texts.HomebrewTimeout()
	case errors.Is(err, brew.ErrInvalidOutput):
		return texts.HomebrewUnexpectedOutput()
	case errors.Is(err, brew.ErrQueryFailed):
		return texts.HomebrewQueryFailed()
	default:
		return texts.UnexpectedError()
	}
}

// checkErrorTooltip explains what the user can do, and falls back to the raw
// error only when the failure could not be classified.
func checkErrorTooltip(texts *localization.Strings, err error) string {
	switch {
	case errors.Is(err, brew.ErrNotFound):
		return texts.HomebrewNotFoundTooltip()
	case errors.Is(err, brew.ErrTimeout):
		return texts.HomebrewTimeoutTooltip()
	case errors.Is(err, brew.ErrInvalidOutput):
		return texts.HomebrewUnexpectedOutputTooltip()
	default:
		// A failed brew run carries its own stderr, which is more useful than
		// any generic sentence Brewtifyer could offer.
		return shortError(err)
	}
}

func packageTitle(texts *localization.Strings, pkg brew.Package) string {
	installed := strings.Join(pkg.InstalledVersions, ", ")
	if installed == "" {
		installed = "?"
	}
	return texts.PackageTitle(pkg.Name, installed, pkg.CurrentVersion, pkg.Pinned)
}

func packageUpdateTooltip(texts *localization.Strings, pkg brew.Package) string {
	if pkg.Pinned {
		return texts.PinnedPackageTooltip()
	}
	return texts.PackageUpgradeTooltip()
}

// shortError keeps a raw error readable in a tooltip. It counts runes rather
// than bytes so that a cut never lands inside a multi-byte character.
func shortError(err error) string {
	const maximumRunes = 200

	message := err.Error()
	if utf8.RuneCountInString(message) <= maximumRunes {
		return message
	}
	return string([]rune(message)[:maximumRunes]) + "…"
}
