// Package brew locates the Homebrew executable, runs update checks, and
// parses the outdated package data. It never depends on localization:
// failures are reported through the sentinel errors in errors.go, which the
// user interface maps to localized text.
package brew

import "time"

// Kind identifies whether an update belongs to a Homebrew formula or cask.
type Kind string

const (
	Formula Kind = "formula"
	Cask    Kind = "cask"
)

// Package describes one installed package for which Homebrew reports an update.
type Package struct {
	Name              string
	Kind              Kind
	InstalledVersions []string
	CurrentVersion    string
	Pinned            bool
}

// WarningKind identifies a non-fatal condition that still produced a usable
// Homebrew result. The UI maps kinds to localized text while Cause remains a
// technical diagnostic for the log.
type WarningKind uint8

const (
	WarningMetadataRefreshFailed WarningKind = iota + 1
)

// Warning describes a non-fatal Homebrew check problem.
type Warning struct {
	Kind  WarningKind
	Cause error
}

// Result is the outcome of one complete Homebrew check.
type Result struct {
	Packages  []Package
	CheckedAt time.Time
	Warning   *Warning
}
