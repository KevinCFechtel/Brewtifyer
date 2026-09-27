## What changed

<!-- Keep the change focused and preserve the Go-first architecture. -->

## Verification

- [ ] `./Build/format.sh`
- [ ] `./Build/vet.sh`
- [ ] `./Build/test.sh` (also validates the localization catalogs)
- [ ] `./Build/build.sh` and the app was launched manually

## Localization

- [ ] No user-facing strings changed, or
- [ ] `internal/localization/messages.go` was updated and both catalogs were
      regenerated and translated (see `CONTRIBUTING.md`)
