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

// Result is the outcome of one complete Homebrew check.
type Result struct {
	Packages  []Package
	CheckedAt time.Time
	Warning   string
}
