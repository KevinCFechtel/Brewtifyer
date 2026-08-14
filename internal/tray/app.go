package tray

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
)

const maxVisibleUpdates = 10

type App struct {
	checker       monitor.Checker
	interval      time.Duration
	resultHandler func(brew.Result)

	ctx     context.Context
	cancel  context.CancelFunc
	monitor *monitor.Monitor
	wait    sync.WaitGroup

	statusItem  *systray.MenuItem
	checkedItem *systray.MenuItem
	updateItems []*systray.MenuItem
	overflow    *systray.MenuItem
	refreshItem *systray.MenuItem
	quitItem    *systray.MenuItem
}

func New(checker monitor.Checker, interval time.Duration, resultHandler func(brew.Result)) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		checker:       checker,
		interval:      interval,
		resultHandler: resultHandler,
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
		item := systray.AddMenuItem("", "")
		item.Disable()
		item.Hide()
		app.updateItems = append(app.updateItems, item)
	}
	app.overflow = systray.AddMenuItem("", "")
	app.overflow.Disable()
	app.overflow.Hide()

	systray.AddSeparator()
	app.refreshItem = systray.AddMenuItem("Jetzt prüfen", "Homebrew sofort auf Updates prüfen")
	app.quitItem = systray.AddMenuItem("Beenden", "Brewtifyer beenden")

	app.monitor = monitor.New(app.checker, app.interval, app.render)
	app.startBackgroundTasks()
}

func (app *App) OnExit() {
	app.cancel()
	app.wait.Wait()
}

func (app *App) startBackgroundTasks() {
	app.wait.Add(3)

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
			item.Hide()
			continue
		}
		item.SetTitle(packageTitle(result.Packages[index]))
		item.Show()
	}

	remaining := count - len(app.updateItems)
	if remaining > 0 {
		app.overflow.SetTitle(fmt.Sprintf("… und %d weitere", remaining))
		app.overflow.Show()
	} else {
		app.overflow.Hide()
	}
}

func (app *App) hideUpdates() {
	for _, item := range app.updateItems {
		item.Hide()
	}
	app.overflow.Hide()
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
