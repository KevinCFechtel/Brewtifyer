# Contributing to Brewtifyer

Thanks for taking the time. Brewtifyer is deliberately small, and the
constraints below are what keep it that way. A change that fits inside them is
much easier to accept.

`AGENTS.md` holds the detailed working notes, including release pitfalls and
icon rules. This file is the short version.

## Before you open a pull request

```sh
./Build/format.sh      # gofmt
./Build/vet.sh         # go vet
./Build/test.sh        # localization check + go test -race ./...
./Build/build.sh       # universal app bundle
open dist/Brewtifyer.app
```

CI runs the same checks plus `golangci-lint` and `govulncheck` on a macOS
runner. Running them locally first is faster than waiting for the workflow.

## Architectural constraints

- **Go-first.** Native code is limited to focused Objective-C/cgo bridges
  under `internal/*/native_darwin.*`.
- **Only `fyne.io/systray`** for the menu bar, not the full Fyne toolkit.
- **Shell scripts in `Build/`**, not a Makefile.
- **The menu bar is reached only through the `Menu` and `MenuItem` interfaces**
  in `internal/tray/menu.go`. Do not call `systray.*` from application code:
  that is what keeps the render logic testable and the library replaceable.
- **Non-darwin implementations stay buildable.** Every package with a
  `native_darwin.go` also has a `native_other.go`.
- **Version metadata lives in `VERSION` and `BUILD_NUMBER`.** Never hard-code a
  release version in Go source or a build script.

## Errors that reach the user

`internal/brew` must not depend on `internal/localization`. Instead, every
error leaving that package wraps one of the sentinels in
`internal/brew/errors.go`, and `checkErrorMessage` in `internal/tray/app.go`
maps them to localized text.

Adding a new failure mode therefore means:

1. Add a sentinel to `internal/brew/errors.go` and wrap it at the failure site.
2. Add a message to `internal/localization/messages.go` and an accessor in
   `internal/localization/strings.go`.
3. Add a case to `checkErrorMessage` and, if useful, `checkErrorTooltip`.
4. Regenerate the catalogs (below).

Never put an English string directly into the menu. `internal/brew/errors_test.go`
asserts that the sentinels are actually wrapped.

## Persisted files must stay forward-compatible

The configuration and the notification state are caches, not sources of truth.
Both must degrade to defaults when they cannot be interpreted, including when
they carry a version this build does not know. Returning an error instead would
mean a newer Brewtifyer permanently breaks an older one after a downgrade.

If you add a persisted file, follow the same rule and cover it with a test like
`TestLoadFallsBackOnUnknownVersion`.

## Localization

English is the source language; German is maintained in
`internal/localization/locales`. After changing any user-facing string:

```sh
go tool goi18n extract -sourceLanguage en -format json \
  -outdir internal/localization/locales internal/localization
go tool goi18n merge -sourceLanguage en -format json \
  -outdir internal/localization/locales \
  internal/localization/locales/active.en.json \
  internal/localization/locales/active.de.json
```

Translate everything in the generated `translate.de.json`, merge it in, delete
the temporary file, and commit both catalogs. German uses the infinitive style
already present in the catalog ("Homebrew installieren", not "Installiere
Homebrew"). `./Build/localization.sh` verifies completeness and normalization.

## Platform floor

The Go toolchain sets the macOS floor of every binary. `APP_DEPLOYMENT_TARGET`
in `Build/version.sh` is the single source of truth for it:

- `Build/build.sh` passes it to the compiler and writes it into
  `LSMinimumSystemVersion` of the generated bundle. The value in
  `Build/Info.plist` is only a placeholder; do not edit it.
- After linking, `Build/build.sh` checks with `vtool` that every architecture
  slice actually targets it, and fails otherwise.

Raising `go`/`toolchain` in `go.mod` can raise the real floor. You will notice,
because the build fails until `APP_DEPLOYMENT_TARGET` is raised too. Update the
README and the cask's `depends_on macos:` in the same change.

## Commits and scope

Keep changes focused. A pull request that fixes one thing and reformats three
files is harder to review than two pull requests.
