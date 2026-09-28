# Changelog

All notable changes to Brewtifyer are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Homebrew is resolved again for every check and interactive upgrade, so a
  running Brewtifyer can recover when Homebrew is installed, moved, or
  replaced without requiring an app restart.
- Non-fatal Homebrew metadata refresh failures are represented as typed
  warnings and localized by the tray layer; technical causes remain in the log.
- The systray lifecycle is fully contained in `internal/tray`, keeping the
  entry point independent of the concrete menu bar library.
- CI pins `golangci-lint` and `govulncheck` to explicit versions so an
  external tool release cannot change the result for an unchanged commit.

### Fixed

- Automatic scheduling now strips Go's monotonic clock component before
  comparing timestamps. On macOS that monotonic clock pauses during system
  sleep, so retaining it could still postpone a six-hour check after wake even
  though the scheduler was intended to use wall-clock time.
- The Homebrew JSON v2 parser now requires the top-level `formulae` and
  `casks` arrays. A future incompatible schema can no longer be mistaken for
  a valid response with zero available updates.
- Raw upgrade and launch-at-login errors are no longer shown directly in menu
  tooltips; localized UI text is used while the technical cause is logged.

## [1.0.1] - 2026-09-27

### Fixed

- The release pipeline signs the build produced by the tagged commit's
  workflow run instead of a local build, and verifies its provenance
  attestation before signing. The cask verification now reads the generated
  file rather than an installed tap clone, which silently passed.

## [1.0.0] - 2026-09-27

### Added

- Optional configuration file at
  `~/Library/Application Support/Brewtifyer/config.json` for the check
  interval, the number of visible package rows, the terminal application, and
  the Homebrew path. Out-of-range values are clamped, and a missing, unreadable
  or future-version file falls back to the defaults.
- File log at `~/Library/Logs/Brewtifyer/brewtifyer.log`, rotated at 1 MiB.
  Previously the log went to stderr only, which is discarded when the app is
  started from Finder or as a login item.
- The Homebrew executable path and version are recorded in the log at startup
  to make bug reports actionable.
- `--help` output describing flags, environment variables, and file locations.
- Universal release binaries covering Apple Silicon and Intel in one download.
- Continuous integration on macOS running formatting, vet, localization checks,
  race-enabled tests, a universal build, `golangci-lint`, and `govulncheck`,
  plus Dependabot updates for Go modules and GitHub Actions.
- `CONTRIBUTING.md`, issue and pull request templates, and this changelog.

### Changed

- **The minimum supported macOS version is now 13.0.** The previously declared
  floor of macOS 11 could not be met: the Go toolchain used to build Brewtifyer
  requires macOS 13, so binaries would not have run reliably on macOS 11 or 12
  despite the bundle advertising support. The Go toolchain is now pinned in
  `go.mod`, `APP_DEPLOYMENT_TARGET` in `Build/version.sh` is the single source
  of truth for the floor, and `Build/build.sh` verifies with `vtool` that every
  architecture slice was actually linked against it. A toolchain upgrade that
  raises the real floor now fails the build instead of shipping a bundle that
  promises an older macOS than it supports.
- Homebrew failures are shown in the menu in the selected language. Previously
  the raw English error text from `internal/brew` appeared underneath an
  otherwise localized menu. Errors are now classified into sentinel values and
  mapped to localized messages, while the technical cause goes to the log.
- Automatic checks are scheduled against the wall clock instead of a single
  long timer. The monotonic clock does not advance while macOS sleeps, so a
  laptop that was closed overnight previously checked hours late while
  presenting stale data as current.
- The notification state and the configuration recover from an unknown schema
  version by starting over rather than failing. Previously a state file written
  by a newer Brewtifyer would disable notifications permanently after a
  downgrade, with no indication in the interface.
- The terminal application used for interactive upgrades is configurable;
  `Terminal` remains the default.
- The menu bar library is reached only through the `Menu` and `MenuItem`
  interfaces, which brought the previously untested render logic under test.
- Launch at login no longer carries a macOS 11/12 fallback path, which the new
  platform floor made unreachable.
- Dependencies updated: `golang.org/x/sys` 0.15.0 → 0.48.0,
  `golang.org/x/text` 0.32.0 → 0.42.0, `github.com/godbus/dbus/v5` 5.1.0 → 5.2.2.
- Build and release scripts now report in English, matching the rest of the
  project's documentation.
- `AGENTS.md` is committed instead of ignored, so the architectural invariants
  and release pitfalls are available to everyone working on the project.
- `Info.plist` declares English as the development region, lists the available
  localizations, and carries a copyright and application category.
- Ad-hoc signing no longer passes `--deep`, which Apple discourages for signing
  and which the bundle does not need.
- Binaries are built with `-trimpath`, so no absolute build-machine path is
  embedded any more. A release build previously shipped 148 references to the
  maintainer's home directory.

### Fixed

- The generated Homebrew cask no longer carries the deprecated `verified:`
  parameter or the string form of `depends_on macos:`. Both made every `brew`
  command that loaded the cask print a deprecation warning, and the string form
  is rejected by `brew style`. The macOS requirement is now derived from
  `APP_DEPLOYMENT_TARGET` instead of spelled out, so it cannot drift from the
  `LSMinimumSystemVersion` that `brew audit --online` reads out of the bundle.
- `zap trash:` lists `~/Library/Preferences/dev.kevincfechtel.Brewtifyer.plist`,
  which AppKit writes for the menu bar item position. `brew uninstall --zap`
  previously left it behind.
- "Check now" is no longer ignored while a check is running. The request was
  dropped whenever it arrived before the running check had finished, so a click
  during the startup check did nothing. It is now queued and served as soon as
  the running check completes, which also removed a timing-dependent test
  failure in CI.
- Background tasks register with the wait group individually instead of relying
  on a hard-coded count, which previously had to be kept in sync by hand and
  would have caused a panic or a hang on the next added task.
- Long error messages are truncated on a rune boundary, so a tooltip can no
  longer end in a broken multi-byte character.

### Added

- First release: menu bar app showing available Homebrew formula and cask
  updates, with automatic Homebrew discovery, six-hour checks, interactive
  upgrades in Terminal, native notifications with deduplication, launch at
  login, English and German localization, and signed, notarized releases.

[Unreleased]: https://github.com/KevinCFechtel/Brewtifyer/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/KevinCFechtel/Brewtifyer/releases/tag/v1.0.1
[1.0.0]: https://github.com/KevinCFechtel/Brewtifyer/releases/tag/v1.0.0
