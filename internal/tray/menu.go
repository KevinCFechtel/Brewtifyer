package tray

import "fyne.io/systray"

// MenuItem is the subset of systray's menu item that Brewtifyer uses.
type MenuItem interface {
	SetTitle(title string)
	SetTooltip(tooltip string)
	Enable()
	Disable()
	Show()
	Hide()
	Check()
	Uncheck()
	AddItem(title, tooltip string) MenuItem
	AddSeparator()
	// Clicked is closed or fed when the user activates the item.
	Clicked() <-chan struct{}
}

// Menu is the subset of the systray package that Brewtifyer uses.
//
// The menu bar library is reached only through this interface. That keeps the
// render logic in app.go testable without a running menu bar, and confines a
// future replacement of fyne.io/systray to the adapter below instead of
// spreading package-level calls through the application.
type Menu interface {
	SetTemplateIcon(icon []byte)
	SetTitle(title string)
	SetTooltip(tooltip string)
	SetRemovalAllowed(allowed bool)
	AddItem(title, tooltip string) MenuItem
	AddCheckbox(title, tooltip string, checked bool) MenuItem
	AddSeparator()
	Quit()
}

// Run owns the production systray lifecycle so no package outside tray needs
// to depend on fyne.io/systray directly.
func Run(app *App) {
	systray.Run(app.OnReady, app.OnExit)
}

// SystrayMenu returns the production menu backed by fyne.io/systray. It may
// only be used from the lifecycle owned by this package.
func SystrayMenu() Menu { return systrayMenu{} }

type systrayMenu struct{}

func (systrayMenu) SetTemplateIcon(icon []byte) {
	// macOS renders a template image in the correct color for the current
	// appearance, so the same bytes serve as both the regular and template icon.
	systray.SetTemplateIcon(icon, icon)
}

func (systrayMenu) SetTitle(title string)              { systray.SetTitle(title) }
func (systrayMenu) SetTooltip(tooltip string)          { systray.SetTooltip(tooltip) }
func (systrayMenu) SetRemovalAllowed(allowed bool)     { systray.SetRemovalAllowed(allowed) }
func (systrayMenu) AddSeparator()                      { systray.AddSeparator() }
func (systrayMenu) Quit()                              { systray.Quit() }
func (systrayMenu) AddItem(title, tip string) MenuItem { return item(systray.AddMenuItem(title, tip)) }

func (systrayMenu) AddCheckbox(title, tooltip string, checked bool) MenuItem {
	return item(systray.AddMenuItemCheckbox(title, tooltip, checked))
}

func item(native *systray.MenuItem) MenuItem { return systrayMenuItem{native: native} }

type systrayMenuItem struct{ native *systray.MenuItem }

func (i systrayMenuItem) SetTitle(title string)     { i.native.SetTitle(title) }
func (i systrayMenuItem) SetTooltip(tooltip string) { i.native.SetTooltip(tooltip) }
func (i systrayMenuItem) Enable()                   { i.native.Enable() }
func (i systrayMenuItem) Disable()                  { i.native.Disable() }
func (i systrayMenuItem) Show()                     { i.native.Show() }
func (i systrayMenuItem) Hide()                     { i.native.Hide() }
func (i systrayMenuItem) Check()                    { i.native.Check() }
func (i systrayMenuItem) Uncheck()                  { i.native.Uncheck() }
func (i systrayMenuItem) AddItem(title, tip string) MenuItem {
	return item(i.native.AddSubMenuItem(title, tip))
}
func (i systrayMenuItem) AddSeparator() { i.native.AddSeparator() }

func (i systrayMenuItem) Clicked() <-chan struct{} { return i.native.ClickedCh }
