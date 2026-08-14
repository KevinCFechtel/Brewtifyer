# Brewtifyer

Brewtifyer is a lightweight, open-source macOS menu bar app that keeps an eye
on Homebrew and lets you know when formula or cask updates are available.

The app is written primarily in Go and uses only the standalone
[`fyne.io/systray`](https://fyne.io/systray) module for its menu bar interface.
Native macOS integrations are kept small and focused.

## Features

- Finds Homebrew automatically on Apple Silicon and Intel Macs.
- Checks formulae and casks immediately after launch and every six hours.
- Refreshes Homebrew metadata only when necessary.
- Shows the number of available updates directly in the menu bar.
- Lists up to ten outdated packages with installed and available versions.
- Opens Terminal to update one selected package or all packages at once.
- Keeps Homebrew interactive and respects pinned packages.
- Sends native macOS notifications for newly discovered package versions.
- Remembers notification state across restarts to avoid duplicates.
- Provides a manual refresh without allowing overlapping checks.
- Supports native launch at login on macOS 13 and later.

## Installation

When release builds are available, download the latest macOS archive from the
[GitHub Releases page](https://github.com/KevinCFechtel/Brewtifyer/releases),
extract it, and move `Brewtifyer.app` to `/Applications`.

Start Brewtifyer from the Applications folder. It runs as a menu bar app and
does not add an icon to the Dock.

## Using Brewtifyer

The menu bar icon displays the current number of available updates. Open its
menu to inspect packages, trigger another check, or quit the app.

Select a package to open Terminal and run its individual Homebrew upgrade. Use
`Alle Updates installieren …` to run `brew upgrade` for all packages. Commands
remain visible and interactive in Terminal, including any prompts produced by
Homebrew.

On macOS 13 or later, enable `Bei Anmeldung starten` to register Brewtifyer as
a login item. If macOS requires approval, Brewtifyer links directly to the
Login Items panel in System Settings. Launch at login is unavailable on macOS
11 and 12.

macOS requests notification permission when Brewtifyer first needs to send an
update notification. Notification deduplication state is stored locally at:

```text
~/Library/Application Support/Brewtifyer/notification-state.json
```

Brewtifyer does not require an account or a separate background service. It
uses the locally installed Homebrew executable for update checks and upgrades.

## Requirements

- macOS 11 or later
- Homebrew
- Go 1.25 or later for building from source
- Xcode Command Line Tools for building from source

## Build from Source

Clone the repository and run:

```sh
./Build/build.sh
open dist/Brewtifyer.app
```

The build script creates an ad-hoc signed development app at
`dist/Brewtifyer.app`.

If Homebrew is installed in a non-standard location, provide the executable
path when starting Brewtifyer:

```sh
BREWTIFYER_BREW_PATH=/path/to/brew ./Build/run.sh
```

## Development

The project intentionally uses shell scripts instead of a Makefile:

```sh
./Build/format.sh
./Build/test.sh
./Build/vet.sh
./Build/run.sh
```

The main packages are organized by responsibility:

- `internal/brew` locates Homebrew, executes checks, and parses outdated data.
- `internal/monitor` schedules checks and prevents overlapping work.
- `internal/tray` owns the menu bar interface and user actions.
- `internal/notification` provides native notifications and deduplication.
- `internal/autostart` manages the native macOS login item.
- `internal/upgrade` opens interactive Homebrew upgrades in Terminal.

The application and menu bar icons can be regenerated with:

```sh
./Build/generate-icons.sh
./Build/generate-menu-bar-icon.sh
```

## Creating a Release

Signed and notarized releases require an Apple Developer ID Application
certificate and a `notarytool` profile stored in the local Keychain:

```sh
xcrun notarytool store-credentials macos-notary
cp Build/.env.example Build/.env
./Build/release.sh
```

`Build/.env` is ignored by Git. The release script verifies the secure
timestamp, code signature, notarization ticket, Gatekeeper acceptance, and the
final extracted archive. Completed archives are written to `dist/release/`.

## Contributing

Issues, bug reports, and pull requests are welcome. Before submitting a change,
please run the formatting, test, and vet scripts listed above. Keep changes
focused and preserve the project's small Go-first architecture.
