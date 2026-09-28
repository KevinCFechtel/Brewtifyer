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
- Keeps the schedule across sleep by comparing wall-clock time, so a closed
  laptop does not delay the next check.
- Refreshes Homebrew metadata only when necessary.
- Shows the number of available updates directly in the menu bar.
- Groups outdated formulae and casks, with installed and available versions.
- Marks updates that appeared or changed since the previous successful check.
- Opens a package submenu for updating or viewing `brew info` in Terminal.
- Updates one package, all formulae, all casks, or everything at once.
- Checks Homebrew again automatically when an interactive update finishes.
- Keeps Homebrew interactive and respects pinned packages.
- Sends native macOS notifications for newly discovered package versions.
- Remembers notification state across restarts to avoid duplicates.
- Provides a manual refresh without allowing overlapping checks.
- Supports native launch at login.
- Uses the operating system language with complete English and German menus,
  notifications, and interactive update messages.
- Reads an optional configuration file for the check interval, the number of
  visible rows, the terminal application, and the Homebrew path.

## Installation

### Homebrew (recommended)

```sh
brew install --cask kevincfechtel/tap/brewtifyer
```

Installing through Homebrew also keeps Brewtifyer itself up to date:
`brew upgrade --cask brewtifyer` picks up new releases like any other cask.

### Manual download

Download the latest macOS archive from the
[GitHub Releases page](https://github.com/KevinCFechtel/Brewtifyer/releases),
extract it, and move `Brewtifyer.app` to `/Applications`. Releases are
universal binaries and run natively on both Apple Silicon and Intel Macs.

Start Brewtifyer from the Applications folder. It runs as a menu bar app and
does not add an icon to the Dock.

## Using Brewtifyer

The menu bar icon displays the current number of available updates. Open its
menu to inspect packages, trigger another check, or quit the app.

Open the Formulae or Casks group and select a package to see its update actions.
Each package submenu can run its individual Homebrew upgrade or show `brew info`
in Terminal. The group actions update all formulae or all casks, while
`Install all updates …` still runs `brew upgrade` for everything.

Interactive update commands remain visible in Terminal, including any prompts
produced by Homebrew. When an update command finishes, Brewtifyer automatically
checks again so the menu reflects the new Homebrew state.

Enable `Launch at login` to register Brewtifyer as a login item. If macOS
requires approval, Brewtifyer links directly to the Login Items panel in
System Settings.

macOS requests notification permission when Brewtifyer first needs to send an
update notification.

Brewtifyer does not require an account or a separate background service. It
uses the locally installed Homebrew executable for update checks and upgrades.

## Configuration

Brewtifyer writes a configuration file on first launch and reads it at every
start:

```text
~/Library/Application Support/Brewtifyer/config.json
```

```json
{
  "version": 1,
  "checkIntervalMinutes": 360,
  "maxVisibleUpdates": 10,
  "terminalApplication": "Terminal",
  "brewPath": ""
}
```

| Setting | Default | Range | Purpose |
| --- | --- | --- | --- |
| `checkIntervalMinutes` | `360` | 15 – 10080 | Time between automatic checks |
| `maxVisibleUpdates` | `10` | 1 – 50 | Package rows shown per Formulae/Casks group |
| `terminalApplication` | `Terminal` | any app name | App used for interactive upgrades, for example `iTerm` or `Ghostty` |
| `brewPath` | `""` | absolute path | Overrides Homebrew autodetection |

The file is optional and never blocks startup. A missing, unreadable, or
future-version file falls back to the defaults, and out-of-range values are
clamped into the accepted range. Every such correction is written to the log.

Changes take effect on the next start. Environment variables still win over
the file for a single run:

```sh
BREWTIFYER_BREW_PATH=/path/to/brew /Applications/Brewtifyer.app/Contents/MacOS/Brewtifyer
BREWTIFYER_LANGUAGE=en /Applications/Brewtifyer.app/Contents/MacOS/Brewtifyer
```

## Logs and state

Brewtifyer is normally started from Finder or as a login item, where nothing is
attached to stderr. It therefore writes its log to a file, rotating it once it
exceeds 1 MiB:

```text
~/Library/Logs/Brewtifyer/brewtifyer.log
```

The log records the Homebrew executable and version, the effective
configuration, and any failure that is only summarized in the menu. It is the
most useful thing to attach to a bug report.

Notification deduplication state is stored separately:

```text
~/Library/Application Support/Brewtifyer/notification-state.json
```

Both the configuration and the notification state are treated as caches. If
either is unreadable or was written by a newer Brewtifyer, it is rebuilt from
defaults instead of blocking the app, so downgrading always works.

## Requirements

- macOS 13 or later
- Homebrew
- Go 1.27 or later for building from source
- Xcode Command Line Tools for building from source

macOS 13 is the floor of the Go toolchain pinned in `go.mod`; a binary built
with it cannot run on earlier versions.

`APP_DEPLOYMENT_TARGET` in `Build/version.sh` is the single source of truth for
that floor. `Build/build.sh` uses it for the compiler, writes it into
`LSMinimumSystemVersion` of the generated bundle, and then verifies with
`vtool` that every architecture slice really was linked against it. A Go
toolchain upgrade that raises the real floor therefore fails the build instead
of shipping a bundle that promises an older macOS than it can run on.

## Build from Source

Clone the repository and run:

```sh
./Build/build.sh
open dist/Brewtifyer.app
```

The build script creates an ad-hoc signed universal development app at
`dist/Brewtifyer.app`. For a faster development loop, build only the local
architecture:

```sh
BREWTIFYER_ARCHS=arm64 ./Build/build.sh
```

Inspect the metadata embedded in a build with:

```sh
dist/Brewtifyer.app/Contents/MacOS/Brewtifyer --version
dist/Brewtifyer.app/Contents/MacOS/Brewtifyer --help
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

Every push and pull request runs formatting, vet, localization, race-enabled
tests, a universal build, `golangci-lint`, and `govulncheck` on a macOS runner.
See `.github/workflows/ci.yml`.

The main packages are organized by responsibility:

- `internal/brew` locates Homebrew, executes checks, and parses outdated data.
- `internal/config` reads the optional user configuration.
- `internal/logging` directs the log to a file the user can find.
- `internal/monitor` schedules checks and prevents overlapping work.
- `internal/tray` owns the menu bar interface and user actions.
- `internal/notification` provides native notifications and deduplication.
- `internal/autostart` manages the native macOS login item.
- `internal/upgrade` opens interactive Homebrew upgrades in a terminal.
- `internal/localization` owns language detection, embedded message catalogs,
  pluralization, and localized date formatting.

The menu bar library is reached only through the `Menu` and `MenuItem`
interfaces in `internal/tray/menu.go`. Keeping it behind an interface makes the
render logic testable without a running menu bar and confines a replacement of
`fyne.io/systray` to a single adapter.

Errors that reach the user interface wrap one of the sentinel errors in
`internal/brew/errors.go`. The menu matches on those to pick a localized
message, while the log keeps the technical cause. A new failure mode needs a
sentinel and an entry in `checkErrorMessage`, not an English string in the UI.

The application and menu bar icons can be regenerated with:

```sh
./Build/generate-icons.sh
./Build/generate-menu-bar-icon.sh
```

The application icon uses an Icon Composer source at `assets/AppIcon.icon` for
the adaptive light, dark, and tinted appearances on current macOS versions.
`Build/Assets.car` contains the compiled icon, while `Build/AppIcon.icns`
remains the fallback. Regenerating these application icon artifacts requires
Xcode 26 or later. The same script also updates
`assets/BrewtifyerIconPreview.png`, which is displayed at the top of this
README. The menu bar template icon is maintained separately and is not part of
the adaptive app icon.

See `CONTRIBUTING.md` for the architectural invariants and `AGENTS.md` for the
detailed working notes.

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
catalogs are complete and match the Go source; CI runs the same check.

At runtime, `BREWTIFYER_LANGUAGE` can override the operating system language,
for example `BREWTIFYER_LANGUAGE=en` or `BREWTIFYER_LANGUAGE=de`.

## Versioning

The platform-independent release version is stored in `VERSION` using the
`MAJOR.MINOR.PATCH` format. `BUILD_NUMBER` contains the positive, monotonically
increasing number of the concrete build. `Build/Info.plist` is only a template;
`Build/build.sh` writes both values into the generated app bundle and embeds
the version, build number, and Git commit in the Go binary.

To prepare a new version, update both files and verify them:

```sh
# Example: VERSION contains 0.2.0 and BUILD_NUMBER contains 2
./Build/version.sh
./Build/test.sh
./Build/vet.sh
```

Record the change in `CHANGELOG.md`, commit it, and create an annotated release
tag that matches `v$(cat VERSION)`:

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

`Build/.env` is ignored by Git. The release script builds a universal binary
and verifies the architecture slices, the secure timestamp, the code signature,
the notarization ticket, Gatekeeper acceptance, version metadata, and the final
extracted archive. Completed archives are written to `dist/release/` as
`Brewtifyer-<version>-macos-universal.zip`.

Signing and notarization stay local because they need private credentials. The
tag workflow in `.github/workflows/release.yml` independently verifies that the
tagged commit builds, tests, and produces both architecture slices.

### Updating the Homebrew cask

The cask lives in the separate `KevinCFechtel/homebrew-tap` repository.
`Build/cask.sh` generates it from the archive that `Build/release.sh` just
produced, so the version and checksum cannot drift from the published artifact:

```sh
./Build/cask.sh                       # print it
./Build/cask.sh ../homebrew-tap/Casks/brewtifyer.rb
```

Upload the release archive to the GitHub release first, then commit the
regenerated cask in the tap repository.

## Contributing

Issues, bug reports, and pull requests are welcome. Please read
`CONTRIBUTING.md` first: it lists the verification steps and the architectural
constraints that keep Brewtifyer small.

## License

MIT. See `LICENSE`.
