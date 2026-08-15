package tray

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
)

const (
	maxVisibleUpdates        = 10
	autostartRefreshInterval = 5 * time.Second
)

type Updater interface {
	UpgradePackage(brew.Package) error
	UpgradeAll() error
}

type App struct {
	checker       monitor.Checker
	interval      time.Duration
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

	statusItem            *systray.MenuItem
	checkedItem           *systray.MenuItem
	updateItems           []*systray.MenuItem
	overflow              *systray.MenuItem
	updateAllItem         *systray.MenuItem
	refreshItem           *systray.MenuItem
	autostartItem         *systray.MenuItem
	autostartSettingsItem *systray.MenuItem
	quitItem              *systray.MenuItem
}

func New(
	checker monitor.Checker,
	interval time.Duration,
	resultHandler func(brew.Result),
	updater Updater,
	autostartController autostart.Controller,
	texts *localization.Strings,
) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		checker:       checker,
		interval:      interval,
		resultHandler: resultHandler,
		updater:       updater,
		autostart:     autostartController,
		texts:         texts,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (app *App) OnReady() {
	icon := iconPNG()
	systray.SetTemplateIcon(icon, icon)
	systray.SetTooltip(app.texts.TrayTooltip())
	systray.SetRemovalAllowed(false)

	app.statusItem = systray.AddMenuItem(app.texts.Checking(), app.texts.CurrentStatusTooltip())
	app.statusItem.Disable()
	app.checkedItem = systray.AddMenuItem(app.texts.NotChecked(), app.texts.LastCheckTooltip())
	app.checkedItem.Disable()
	systray.AddSeparator()

	for range maxVisibleUpdates {
		item := systray.AddMenuItem("", app.texts.UpgradePackageMenuTooltip())
		item.Disable()
		item.Hide()
		app.updateItems = append(app.updateItems, item)
	}
	app.overflow = systray.AddMenuItem("", "")
	app.overflow.Disable()
	app.overflow.Hide()
	app.updateAllItem = systray.AddMenuItem(app.texts.UpgradeAll(), app.texts.UpgradeAllTooltip())
	app.updateAllItem.Disable()
	app.updateAllItem.Hide()

	systray.AddSeparator()
	app.refreshItem = systray.AddMenuItem(app.texts.Refresh(), app.texts.RefreshTooltip())
	app.autostartItem = systray.AddMenuItemCheckbox(app.texts.AutostartTitle(), app.texts.AutostartEnableTooltip(), false)
	app.autostartSettingsItem = systray.AddMenuItem(
		app.texts.OpenLoginItems(),
		app.texts.OpenLoginItemsTooltip(),
	)
	app.autostartSettingsItem.Hide()
	systray.AddSeparator()
	app.quitItem = systray.AddMenuItem(app.texts.Quit(), app.texts.QuitTooltip())

	app.monitor = monitor.New(app.checker, app.interval, app.render)
	app.refreshAutostart()
	app.startBackgroundTasks()
}

func (app *App) OnExit() {
	app.cancel()
	app.wait.Wait()
}

func (app *App) startBackgroundTasks() {
	app.wait.Add(5 + len(app.updateItems))

	go func() {
		defer app.wait.Done()
		app.monitor.Run(app.ctx)
	}()

	go func() {
		defer app.wait.Done()
		for {
			select {
			case <-app.ctx.Done():
				return
			case <-app.refreshItem.ClickedCh:
				app.monitor.Trigger()
			}
		}
	}()

	go func() {
		defer app.wait.Done()
		select {
		case <-app.ctx.Done():
			return
		case <-app.quitItem.ClickedCh:
			app.cancel()
			systray.Quit()
		}
	}()

	for index, item := range app.updateItems {
		go func() {
			defer app.wait.Done()
			for {
				select {
				case <-app.ctx.Done():
					return
				case _, open := <-item.ClickedCh:
					if !open {
						return
					}
					app.upgradePackage(index)
				}
			}
		}()
	}

	go func() {
		defer app.wait.Done()
		for {
			select {
			case <-app.ctx.Done():
				return
			case _, open := <-app.updateAllItem.ClickedCh:
				if !open {
					return
				}
				app.upgradeAll()
			}
		}
	}()

	go func() {
		defer app.wait.Done()
		ticker := time.NewTicker(autostartRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-app.ctx.Done():
				return
			case _, open := <-app.autostartItem.ClickedCh:
				if !open {
					return
				}
				app.toggleAutostart()
			case _, open := <-app.autostartSettingsItem.ClickedCh:
				if !open {
					return
				}
				app.openAutostartSettings()
			case <-ticker.C:
				app.refreshAutostart()
			}
		}
	}()
}

func (app *App) render(state monitor.State) {
	if state.Checking {
		systray.SetTitle("…")
		app.statusItem.SetTitle(app.texts.Checking())
		app.refreshItem.Disable()
		return
	}

	app.refreshItem.Enable()
	if state.Err != nil {
		systray.SetTitle("!")
		app.statusItem.SetTitle(app.texts.CheckFailed())
		app.checkedItem.SetTitle(shortError(state.Err))
		app.checkedItem.SetTooltip(state.Err.Error())
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
		systray.SetTitle("")
		app.statusItem.SetTitle(app.texts.UpToDate())
	} else {
		systray.SetTitle(fmt.Sprintf("%d", count))
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

func shortError(err error) string {
	message := err.Error()
	const maximumLength = 90
	if len(message) <= maximumLength {
		return message
	}
	return message[:maximumLength] + "…"
}
