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
) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		checker:       checker,
		interval:      interval,
		resultHandler: resultHandler,
		updater:       updater,
		autostart:     autostartController,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (app *App) OnReady() {
	icon := iconPNG()
	systray.SetTemplateIcon(icon, icon)
	systray.SetTooltip("Brewtifyer – Homebrew Updates")
	systray.SetRemovalAllowed(false)

	app.statusItem = systray.AddMenuItem("Homebrew wird geprüft …", "Aktueller Homebrew-Status")
	app.statusItem.Disable()
	app.checkedItem = systray.AddMenuItem("Noch nicht geprüft", "Zeitpunkt der letzten Prüfung")
	app.checkedItem.Disable()
	systray.AddSeparator()

	for range maxVisibleUpdates {
		item := systray.AddMenuItem("", "Im Terminal aktualisieren")
		item.Disable()
		item.Hide()
		app.updateItems = append(app.updateItems, item)
	}
	app.overflow = systray.AddMenuItem("", "")
	app.overflow.Disable()
	app.overflow.Hide()
	app.updateAllItem = systray.AddMenuItem("Alle Updates installieren …", "brew upgrade im Terminal ausführen")
	app.updateAllItem.Disable()
	app.updateAllItem.Hide()

	systray.AddSeparator()
	app.refreshItem = systray.AddMenuItem("Jetzt prüfen", "Homebrew sofort auf Updates prüfen")
	app.autostartItem = systray.AddMenuItemCheckbox("Bei Anmeldung starten", "Brewtifyer automatisch starten", false)
	app.autostartSettingsItem = systray.AddMenuItem(
		"Anmeldeobjekte in Systemeinstellungen öffnen …",
		"Autostart für Brewtifyer in macOS freigeben",
	)
	app.autostartSettingsItem.Hide()
	systray.AddSeparator()
	app.quitItem = systray.AddMenuItem("Beenden", "Brewtifyer beenden")

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
		app.statusItem.SetTitle("Homebrew wird geprüft …")
		app.refreshItem.Disable()
		return
	}

	app.refreshItem.Enable()
	if state.Err != nil {
		systray.SetTitle("!")
		app.statusItem.SetTitle("Fehler bei der Prüfung")
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
		app.statusItem.SetTitle("Homebrew ist aktuell")
	} else {
		systray.SetTitle(fmt.Sprintf("%d", count))
		app.statusItem.SetTitle(updateCountTitle(count))
	}

	checkedTitle := "Zuletzt geprüft: " + result.CheckedAt.Format("02.01.2006, 15:04")
	if result.Warning != "" {
		checkedTitle += " (mit Warnung)"
		app.checkedItem.SetTooltip(result.Warning)
	} else {
		app.checkedItem.SetTooltip("Zeitpunkt der letzten erfolgreichen Prüfung")
	}
	app.checkedItem.SetTitle(checkedTitle)

	for index, item := range app.updateItems {
		if index >= count {
			item.Disable()
			item.Hide()
			continue
		}
		item.SetTitle(packageTitle(result.Packages[index]))
		item.SetTooltip(packageUpdateTooltip(result.Packages[index]))
		if app.updater != nil {
			item.Enable()
		} else {
			item.Disable()
		}
		item.Show()
	}

	remaining := count - len(app.updateItems)
	if remaining > 0 {
		app.overflow.SetTitle(fmt.Sprintf("… und %d weitere", remaining))
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
	app.statusItem.SetTitle("Update läuft im Terminal …")
}

func (app *App) upgradeAll() {
	if app.updater == nil {
		return
	}
	if err := app.updater.UpgradeAll(); err != nil {
		app.reportUpgradeError(err)
		return
	}
	app.statusItem.SetTitle("Updates laufen im Terminal …")
}

func (app *App) reportUpgradeError(err error) {
	log.Printf("Homebrew-Update konnte nicht gestartet werden: %v", err)
	app.statusItem.SetTitle("Update-Terminal konnte nicht geöffnet werden")
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
		app.applyAutostartMenuState(autostartMenuStateFor(autostart.Unsupported))
		return
	}
	status, err := app.autostart.Status()
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(status))
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
		app.applyAutostartMenuState(autostartMenuStateFor(status))
		return
	}

	resultingStatus, err := app.autostart.SetEnabled(desiredEnabled)
	if err != nil {
		app.reportAutostartError(err)
		return
	}
	app.applyAutostartMenuState(autostartMenuStateFor(resultingStatus))
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
	log.Printf("Autostart konnte nicht verwaltet werden: %v", err)
	app.autostartItem.SetTitle("Autostart konnte nicht geändert werden")
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

func autostartMenuStateFor(status autostart.Status) autostartMenuState {
	switch status {
	case autostart.Disabled:
		return autostartMenuState{
			title:   "Bei Anmeldung starten",
			tooltip: "Brewtifyer automatisch nach der Anmeldung starten",
			enabled: true,
		}
	case autostart.Enabled:
		return autostartMenuState{
			title:   "Bei Anmeldung starten",
			tooltip: "Autostart für Brewtifyer deaktivieren",
			checked: true,
			enabled: true,
		}
	case autostart.RequiresApproval:
		return autostartMenuState{
			title:        "Bei Anmeldung starten (Freigabe erforderlich)",
			tooltip:      "In den macOS-Systemeinstellungen freigeben",
			enabled:      true,
			showSettings: true,
		}
	case autostart.NotFound:
		return autostartMenuState{
			title:   "Bei Anmeldung starten",
			tooltip: "Brewtifyer als Anmeldeobjekt registrieren",
			enabled: true,
		}
	default:
		return autostartMenuState{
			title:   "Bei Anmeldung starten (ab macOS 13)",
			tooltip: "Diese Funktion benötigt macOS 13 oder neuer",
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

func packageTitle(pkg brew.Package) string {
	installed := strings.Join(pkg.InstalledVersions, ", ")
	if installed == "" {
		installed = "?"
	}
	pinned := ""
	if pkg.Pinned {
		pinned = " · angeheftet"
	}
	return fmt.Sprintf("%s: %s → %s%s", pkg.Name, installed, pkg.CurrentVersion, pinned)
}

func packageUpdateTooltip(pkg brew.Package) string {
	if pkg.Pinned {
		return "Im Terminal öffnen; Homebrew überspringt angeheftete Pakete"
	}
	return "Dieses Paket im Terminal aktualisieren"
}

func updateCountTitle(count int) string {
	if count == 1 {
		return "1 Update verfügbar"
	}
	return fmt.Sprintf("%d Updates verfügbar", count)
}

func shortError(err error) string {
	message := err.Error()
	const maximumLength = 90
	if len(message) <= maximumLength {
		return message
	}
	return message[:maximumLength] + "…"
}
