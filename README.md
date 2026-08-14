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
- opens individual package upgrades or `brew upgrade` for all packages in Terminal;
- sends native macOS notifications for newly available package versions;
- persists the last update set to avoid duplicate notifications across restarts;
- offers a native launch-at-login toggle on macOS 13 and later;
- allows a manual refresh without overlapping checks.

Preferences are planned next.

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

macOS asks for notification permission when Brewtifyer first finds an update.
The deduplication state is stored in
`~/Library/Application Support/Brewtifyer/notification-state.json`.

Select an update in the menu to run it individually in Terminal, or choose
`Alle Updates installieren …` to run `brew upgrade`. Homebrew remains
interactive and keeps its normal confirmation prompts. Pinned packages remain
pinned and are skipped by Homebrew.

On macOS 13 or later, use `Bei Anmeldung starten` in the menu to register
Brewtifyer as a login item. If macOS requires approval, Brewtifyer links
directly to the Login Items panel in System Settings. The menu entry remains
disabled on macOS 11 and 12.

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
