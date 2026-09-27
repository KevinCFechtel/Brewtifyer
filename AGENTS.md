# Brewtifyer Agent Notes

These notes are committed on purpose: they are the project's institutional
memory. `CONTRIBUTING.md` is the shorter, contributor-facing version.
Local-only notes belong in `AGENTS.local.md`, which is ignored.

## Product and platform

- Brewtifyer is a macOS menu bar app for Homebrew formula and cask updates.
- The public documentation is in English. The application is fully localized
  in English and German and follows the operating system language.
- The bundle identifier is `dev.kevincfechtel.Brewtifyer`.
- The deployment target is macOS 13. It is dictated by the Go toolchain:
  Go 1.27 does not support earlier macOS versions. `go`/`toolchain` in
  `go.mod`, `MACOSX_DEPLOYMENT_TARGET` in `Build/build.sh`, and
  `LSMinimumSystemVersion` in `Build/Info.plist` must always agree.
- Because the floor is macOS 13, `SMAppService` is always available and the
  native login-item code needs no `@available` fallback.
  `autostart.Unsupported` remains only for the non-darwin build.
- Go 1.27 is pinned in `go.mod` so the macOS floor cannot move silently.

## Architectural constraints

- Keep the application Go-first.
- Use only the standalone `fyne.io/systray` module. Do not add the full Fyne
  toolkit unless the user explicitly changes this constraint.
- Reach the menu bar only through the `Menu` and `MenuItem` interfaces in
  `internal/tray/menu.go`. Application code must not call `systray.*`
  directly; that is what keeps the render logic testable.
- Use shell scripts in `Build/`; do not introduce a Makefile.
- Keep native macOS code limited to focused Objective-C/cgo bridges.
- Native notification code lives in `internal/notification/native_darwin.*`.
- Native login-item code lives in `internal/autostart/native_darwin.*` and uses
  `SMAppService.mainAppService`.
- Localization uses `github.com/nicksnyder/go-i18n/v2`. Typed messages,
  language detection, and embedded catalogs live in `internal/localization`.
- English is the source and fallback language. Keep the committed English and
  German catalogs complete and normalized.
- Version metadata is owned by the repository-root `VERSION` and
  `BUILD_NUMBER` files. Do not duplicate hard-coded release versions in Go or
  build scripts.
- Preserve non-darwin implementations so packages remain buildable elsewhere.

## Behavior that must remain intact

- Locate Homebrew on Apple Silicon and Intel, with
  `BREWTIFYER_BREW_PATH` as an explicit override.
- Parse `brew outdated --json=v2` for formulae and casks.
- Check immediately and then on the configured interval (six hours by
  default) without overlapping checks. Scheduling compares wall-clock time,
  because the monotonic clock stops while macOS sleeps.
- Display at most `maxVisibleUpdates` package rows (ten by default) and
  summarize the remainder in the overflow row.
- Open upgrades in the configured terminal application (Terminal by
  default) so Homebrew remains visible and interactive.
- Support both individual upgrades and `brew upgrade` for all packages.
- Keep native notification deduplication across restarts. State is stored at
  `~/Library/Application Support/Brewtifyer/notification-state.json`.
- Treat `autostart.NotFound` as registerable: the menu must stay enabled and a
  click must call `SetEnabled(true)`. Only `Unsupported` disables the feature.
- If macOS reports `RequiresApproval`, expose the System Settings Login Items
  action.
- Follow the operating system language and support `BREWTIFYER_LANGUAGE` as an
  explicit `en` or `de` override.
- Never show a raw Go error in the menu. Errors from `internal/brew` wrap the
  sentinels in `internal/brew/errors.go`; `checkErrorMessage` maps them to
  localized text and the log keeps the technical cause.
- Persisted files (`config.json`, `notification-state.json`) are caches. An
  unreadable or unknown-version file must fall back to defaults rather than
  returning an error, so that downgrading never wedges the app.

## Build and verification

- Use `./Build/format.sh`, `./Build/localization.sh`, `./Build/test.sh`, and
  `./Build/vet.sh` for normal verification. The test script also validates the
  localization catalogs.
- `./Build/build.sh` creates an ad-hoc signed universal development bundle and
  overwrites `dist/Brewtifyer.app`. Do not run it after a release if the signed
  bundle in `dist` must be preserved.
- `BREWTIFYER_ARCHS=arm64 ./Build/build.sh` builds a single slice for a faster
  development loop. Releases must stay universal.
- `./Build/build.sh` reads `VERSION` and `BUILD_NUMBER`, writes both values to
  the bundle, and embeds version, build number, and Git commit in the binary.
- `./Build/version.sh` validates version metadata. `VERSION` must use
  `MAJOR.MINOR.PATCH`; `BUILD_NUMBER` must be a positive, monotonically
  increasing integer.
- The duplicate `-lobjc` linker warning can occur because systray and multiple
  native bridges use Objective-C; it is currently benign.
- Preserve unrelated user changes in a dirty worktree.

## Continuous integration

- `.github/workflows/ci.yml` runs gofmt verification, vet, the localization
  check, `go test -race`, a universal build, `golangci-lint`, and `govulncheck`
  on a macOS runner. Keep the scripts CI-usable.
- `Build/format.sh` rewrites files, so CI asserts with `gofmt -l` instead.
- `.github/workflows/release.yml` verifies a tagged commit and uploads an
  unsigned bundle. Signing and notarization stay local by design.
- Dependabot updates Go modules weekly and GitHub Actions monthly.

## Release process

- `./Build/release.sh` creates a fresh universal build itself before it signs,
  notarizes, staples, verifies Gatekeeper, archives, extracts, and verifies the
  final artifact again. Running `build.sh` first is unnecessary.
- The release verifies that both the arm64 and x86_64 slices are present.
  Archives are named `Brewtifyer-<version>-macos-universal.zip`.
- If the current commit already has a `v*` release tag, it must include
  `v$(cat VERSION)`. Rebuilding the same app version requires increasing
  `BUILD_NUMBER`.
- Releases require `SIGNING_IDENTITY` and `NOTARY_PROFILE` from `Build/.env`.
- Never commit `Build/.env`, signing certificates, API keys, or notary secrets.
- The Apple RFC-3161 timestamp POST stalls over IPv6 on the current development
  network while IPv4 succeeds. The release script dynamically resolves the
  current IPv4 address for `timestamp.apple.com` and uses `/ts01` for signing.
- Keep the explicit `Timestamp=` validation after signing. Do not weaken the
  signature, notarization, stapler, Gatekeeper, or extracted-archive checks.
- Do not run the release script merely as a test: it uses private signing
  credentials and submits an external notarization request.

## Assets

- The modern application icon source is `assets/AppIcon.icon`. It is a layered
  Icon Composer asset with separate cup, coffee, and steam vectors.
- Keep the coffee layer in front of the cup so the ellipse remains visible.
- The light appearance uses the orange brand background. The dark appearance
  uses Apple's native `system-dark`; keep the cup and steam light and the
  coffee dark brown.
- `Build/Assets.car` contains the adaptive light, dark, and tinted icon for
  current macOS. `assets/BrewtifyerIcon.png` and `Build/AppIcon.icns` remain
  the legacy source and fallback.
- Regenerate `Build/Assets.car`, `Build/AppIcon.icns`, and
  `assets/BrewtifyerIconPreview.png` with `./Build/generate-icons.sh`. Icon
  Composer compilation requires Xcode 26 or later.
- The README displays `assets/BrewtifyerIconPreview.png`; keep it generated
  from the adaptive color icon instead of editing it independently.
- The menu bar source is `internal/tray/menu_bar_icon.svg`; regenerate the PNG
  with `./Build/generate-menu-bar-icon.sh`.
- The menu bar icon must remain a monochrome macOS template image with a
  transparent background. It is intentionally independent of the adaptive app
  icon and must not be replaced with the color icon.
