module github.com/KevinCFechtel/Brewtifyer

// The Go toolchain determines the macOS floor of every produced binary:
// Go 1.27 requires macOS 13 or later. Keep LSMinimumSystemVersion in
// Build/Info.plist and MACOSX_DEPLOYMENT_TARGET in Build/build.sh in sync
// with this line.
go 1.27

toolchain go1.27.1

tool github.com/nicksnyder/go-i18n/v2/goi18n

require (
	fyne.io/systray v1.12.2
	github.com/nicksnyder/go-i18n/v2 v2.6.1
	golang.org/x/sys v0.48.0
	golang.org/x/text v0.42.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
)
