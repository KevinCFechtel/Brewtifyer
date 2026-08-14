package tray

import _ "embed"

//go:embed menu_bar_icon.png
var menuBarIconPNG []byte

// iconPNG returns the monochrome template icon used by the macOS menu bar.
// macOS applies the correct foreground color for light and dark appearances.
func iconPNG() []byte {
	return menuBarIconPNG
}
