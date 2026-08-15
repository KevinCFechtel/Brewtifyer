# Brewtifyer

<p align="center">
  <img src="assets/BrewtifyerIconPreview.png" alt="Brewtifyer app icon" width="192">
</p>

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
- Uses the operating system language with complete English and German menus,
  notifications, and interactive update messages.

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
./Build/localization.sh
./Build/test.sh
./Build/vet.sh
./Build/version.sh
./Build/run.sh
```

The main packages are organized by responsibility:

- `internal/brew` locates Homebrew, executes checks, and parses outdated data.
- `internal/monitor` schedules checks and prevents overlapping work.
- `internal/tray` owns the menu bar interface and user actions.
- `internal/notification` provides native notifications and deduplication.
- `internal/autostart` manages the native macOS login item.
- `internal/upgrade` opens interactive Homebrew upgrades in Terminal.
- `internal/localization` owns language detection, embedded message catalogs,
  pluralization, and localized date formatting.

The application and menu bar icons can be regenerated with:

```sh
./Build/generate-icons.sh
./Build/generate-menu-bar-icon.sh
```

The application icon uses an Icon Composer source at `assets/AppIcon.icon` for
the adaptive light, dark, and tinted appearances on current macOS versions.
`Build/Assets.car` contains the compiled icon, while `Build/AppIcon.icns`
remains the fallback for older supported macOS versions. Regenerating these
application icon artifacts requires Xcode 26 or later. The same script also
updates `assets/BrewtifyerIconPreview.png`, which is displayed at the top of
this README. The menu bar template icon is maintained separately and is not
part of the adaptive app icon.

## Localization

English is Brewtifyer's source and fallback language. German is maintained in
the embedded JSON catalogs under `internal/localization/locales`.

User-facing messages are defined as typed methods in `internal/localization`.
After adding or changing a message, use the pinned `goi18n` tool to extract the
English catalog and merge the German translation:

```sh
go tool goi18n extract -sourceLanguage en -format json \
  -outdir internal/localization/locales internal/localization
go tool goi18n merge -sourceLanguage en -format json \
  -outdir internal/localization/locales \
  internal/localization/locales/active.en.json \
  internal/localization/locales/active.de.json
```

Translate every entry written to `translate.de.json`, merge it again, and
commit the resulting `active.de.json`. The temporary translation file can then
be removed. Run `./Build/localization.sh` to verify that both committed
catalogs are complete and match the Go source.

At runtime, `BREWTIFYER_LANGUAGE` can override the operating system language,
for example `BREWTIFYER_LANGUAGE=en` or `BREWTIFYER_LANGUAGE=de`.

## Versioning

The platform-independent release version is stored in `VERSION` using the
`MAJOR.MINOR.PATCH` format. `BUILD_NUMBER` contains the positive, monotonically
increasing number of the concrete build. `Build/Info.plist` is only a template;
`Build/build.sh` writes both values into the generated app bundle and embeds
the version, build number, and Git commit in the Go binary.

The metadata embedded in a built binary can be inspected with:

```sh
dist/Brewtifyer.app/Contents/MacOS/Brewtifyer --version
```

To prepare a new version, update both files and verify them:

```sh
# Example: VERSION contains 0.2.0 and BUILD_NUMBER contains 2
./Build/version.sh
./Build/test.sh
./Build/vet.sh
```

Commit the version change and optionally create an annotated release tag that
matches `v$(cat VERSION)`, for example:

```sh
git tag -a v0.2.0 -m "Brewtifyer 0.2.0"
```

If the current commit already has a tag beginning with `v`, the release script
requires it to match `VERSION`. Rebuilding the same release version is possible
by keeping `VERSION` and increasing only `BUILD_NUMBER`.

## Creating a Release

Signed and notarized releases require an Apple Developer ID Application
certificate and a `notarytool` profile stored in the local Keychain:

```sh
xcrun notarytool store-credentials macos-notary
cp Build/.env.example Build/.env
./Build/release.sh
```

`Build/.env` is ignored by Git. The release script verifies the secure
timestamp, code signature, notarization ticket, Gatekeeper acceptance, version
metadata, and the final extracted archive. Completed archives are written to
`dist/release/`.

## Contributing

Issues, bug reports, and pull requests are welcome. Before submitting a change,
please run the formatting, test, and vet scripts listed above. Keep changes
focused and preserve the project's small Go-first architecture.
