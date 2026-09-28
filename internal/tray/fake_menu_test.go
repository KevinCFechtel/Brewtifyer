package tray

import (
	"sync"
	"testing"
)

// fakeMenu records everything the application does to the menu bar so that the
// render logic can be asserted without a running systray.
type fakeMenu struct {
	mutex       sync.Mutex
	title       string
	tooltip     string
	icon        []byte
	quitCalls   int
	separators  int
	items       []*fakeMenuItem
	removalFlag bool
}

func newFakeMenu() *fakeMenu { return &fakeMenu{} }

func (menu *fakeMenu) SetTemplateIcon(icon []byte) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.icon = icon
}

func (menu *fakeMenu) SetTitle(title string) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.title = title
}

func (menu *fakeMenu) SetTooltip(tooltip string) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.tooltip = tooltip
}

func (menu *fakeMenu) SetRemovalAllowed(allowed bool) {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.removalFlag = allowed
}

func (menu *fakeMenu) AddSeparator() {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.separators++
}

func (menu *fakeMenu) Quit() {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	menu.quitCalls++
}

func (menu *fakeMenu) AddItem(title, tooltip string) MenuItem {
	return menu.add(title, tooltip, false)
}

func (menu *fakeMenu) AddCheckbox(title, tooltip string, checked bool) MenuItem {
	return menu.add(title, tooltip, checked)
}

func (menu *fakeMenu) add(title, tooltip string, checked bool) MenuItem {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()

	created := &fakeMenuItem{
		title:   title,
		tooltip: tooltip,
		checked: checked,
		// Enabled and visible match systray's defaults for a new item.
		enabled: true,
		visible: true,
		clicks:  make(chan struct{}, 1),
	}
	menu.items = append(menu.items, created)
	return created
}

func (menu *fakeMenu) currentTitle() string {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	return menu.title
}

func (menu *fakeMenu) quitCount() int {
	menu.mutex.Lock()
	defer menu.mutex.Unlock()
	return menu.quitCalls
}

type fakeMenuItem struct {
	mutex      sync.Mutex
	title      string
	tooltip    string
	enabled    bool
	visible    bool
	checked    bool
	clicks     chan struct{}
	children   []*fakeMenuItem
	separators int
}

func (item *fakeMenuItem) SetTitle(title string) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.title = title
}

func (item *fakeMenuItem) SetTooltip(tooltip string) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.tooltip = tooltip
}

func (item *fakeMenuItem) Enable() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.enabled = true
}

func (item *fakeMenuItem) Disable() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.enabled = false
}

func (item *fakeMenuItem) Show() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.visible = true
}

func (item *fakeMenuItem) Hide() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.visible = false
}

func (item *fakeMenuItem) Check() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.checked = true
}

func (item *fakeMenuItem) Uncheck() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.checked = false
}

func (item *fakeMenuItem) AddItem(title, tooltip string) MenuItem {
	item.mutex.Lock()
	defer item.mutex.Unlock()

	child := &fakeMenuItem{
		title:   title,
		tooltip: tooltip,
		enabled: true,
		visible: true,
		clicks:  make(chan struct{}, 1),
	}
	item.children = append(item.children, child)
	return child
}

func (item *fakeMenuItem) AddSeparator() {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	item.separators++
}

func (item *fakeMenuItem) Clicked() <-chan struct{} { return item.clicks }

func (item *fakeMenuItem) click() { item.clicks <- struct{}{} }

func (item *fakeMenuItem) state() (title, tooltip string, enabled, visible, checked bool) {
	item.mutex.Lock()
	defer item.mutex.Unlock()
	return item.title, item.tooltip, item.enabled, item.visible, item.checked
}

func (item *fakeMenuItem) currentTitle() string {
	title, _, _, _, _ := item.state()
	return title
}

func (item *fakeMenuItem) isVisible() bool {
	_, _, _, visible, _ := item.state()
	return visible
}

func (item *fakeMenuItem) isEnabled() bool {
	_, _, enabled, _, _ := item.state()
	return enabled
}

// fakeItem unwraps a MenuItem the application stored as an interface value.
func fakeItem(t *testing.T, item MenuItem) *fakeMenuItem {
	t.Helper()

	concrete, ok := item.(*fakeMenuItem)
	if !ok {
		t.Fatalf("menu item is %T, want *fakeMenuItem", item)
	}
	return concrete
}

func visible(t *testing.T, item MenuItem) bool {
	t.Helper()
	return fakeItem(t, item).isVisible()
}

func enabled(t *testing.T, item MenuItem) bool {
	t.Helper()
	return fakeItem(t, item).isEnabled()
}

func titleOf(t *testing.T, item MenuItem) string {
	t.Helper()
	return fakeItem(t, item).currentTitle()
}

func tooltipOf(t *testing.T, item MenuItem) string {
	t.Helper()
	_, tooltip, _, _, _ := fakeItem(t, item).state()
	return tooltip
}

func childOf(t *testing.T, item MenuItem, index int) *fakeMenuItem {
	t.Helper()
	parent := fakeItem(t, item)
	parent.mutex.Lock()
	defer parent.mutex.Unlock()
	if index < 0 || index >= len(parent.children) {
		t.Fatalf("child index %d out of range (len=%d)", index, len(parent.children))
	}
	return parent.children[index]
}
