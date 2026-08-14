# Brewtifyer

Brewtifyer is an open-source macOS menu bar app that monitors Homebrew for
available formula and cask updates.

The application intentionally uses the standalone [`fyne.io/systray`](https://fyne.io/systray)
module instead of the full Fyne GUI toolkit.

## Current status

The first vertical slice is implemented:

- discovers Homebrew on Apple Silicon and Intel Macs;
- refreshes Homebrew metadata when needed;
- parses `brew outdated --json=v2`;
- checks immediately and every six hours;
- displays up to ten available updates in the menu bar;
- allows a manual refresh without overlapping checks.

Desktop notifications, preferences, and login item support are planned next.

## Requirements

- macOS 11 or later
- Go 1.25 or later
- Xcode Command Line Tools
- Homebrew

## Build the App

```sh
./Build/build.sh
open dist/Brewtifyer.app
```

The build script creates an ad-hoc signed `dist/Brewtifyer.app` without a Dock
icon. If Homebrew is installed in a non-standard location, set
`BREWTIFYER_BREW_PATH` to its executable before starting Brewtifyer.

## Create a Release

A signed and notarized release requires an Apple Developer ID and a
`notarytool` profile stored locally in the Keychain:

```sh
xcrun notarytool store-credentials macos-notary
cp Build/.env.example Build/.env
./Build/release.sh
```

`Build/.env` is ignored by Git. The completed archive is written to
`dist/release/`.

The committed app icon can be regenerated from `assets/BrewtifyerIcon.png`
with `./Build/generate-icons.sh`.

The monochrome menu bar icon can be regenerated from
`internal/tray/menu_bar_icon.svg` with `./Build/generate-menu-bar-icon.sh`.

## Development

```sh
./Build/format.sh
./Build/test.sh
./Build/vet.sh
./Build/run.sh
```
