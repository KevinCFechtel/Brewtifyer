// Package config owns Brewtifyer's user-editable settings. The file is optional:
// a missing, unreadable, or unknown-version file always yields working defaults
// so that a bad or future config can never keep the app from starting.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Version is the schema version written to disk. Bump it only for a change that
// an older Brewtifyer cannot interpret safely; additive fields do not need it.
const Version = 1

// Defaults. CheckInterval is deliberately long: Homebrew metadata changes
// slowly and every check spawns two brew processes.
const (
	DefaultCheckInterval       = 6 * time.Hour
	DefaultMaxVisibleUpdates   = 10
	DefaultTerminalApplication = "Terminal"
)

// Accepted ranges. Values outside them are clamped rather than rejected, so a
// typo degrades the setting instead of the app.
const (
	MinimumCheckInterval     = 15 * time.Minute
	MaximumCheckInterval     = 7 * 24 * time.Hour
	MinimumMaxVisibleUpdates = 1
	MaximumMaxVisibleUpdates = 50
)

// Config holds the effective settings for one Brewtifyer process.
type Config struct {
	// CheckInterval is the wall-clock time between automatic checks.
	CheckInterval time.Duration
	// MaxVisibleUpdates caps the package rows rendered in each package group.
	MaxVisibleUpdates int
	// TerminalApplication is the app opened for interactive upgrades.
	TerminalApplication string
	// BrewPath overrides Homebrew autodetection when not empty.
	BrewPath string
}

// file mirrors the on-disk schema. Every setting is a pointer so that an absent
// field falls back to its default instead of to the Go zero value.
type file struct {
	Version              *int    `json:"version"`
	CheckIntervalMinutes *int    `json:"checkIntervalMinutes"`
	MaxVisibleUpdates    *int    `json:"maxVisibleUpdates"`
	TerminalApplication  *string `json:"terminalApplication"`
	BrewPath             *string `json:"brewPath"`
}

// Default returns the settings used when no config file exists.
func Default() Config {
	return Config{
		CheckInterval:       DefaultCheckInterval,
		MaxVisibleUpdates:   DefaultMaxVisibleUpdates,
		TerminalApplication: DefaultTerminalApplication,
	}
}

// DefaultPath returns the canonical config location next to the notification
// state, so that everything Brewtifyer persists lives in one directory.
func DefaultPath() (string, error) {
	configurationDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user configuration directory could not be determined: %w", err)
	}
	return filepath.Join(configurationDirectory, "Brewtifyer", "config.json"), nil
}

// Load reads the config file at path. It never fails on content: anything that
// cannot be interpreted falls back to the default for that setting and is
// reported through notes, which the caller is expected to log. A missing file
// is written back with the defaults so that the settings are discoverable.
func Load(path string) (Config, []string) {
	configuration := Default()

	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if writeErr := Save(path, configuration); writeErr != nil {
			return configuration, []string{fmt.Sprintf(
				"default configuration could not be written to %s: %v", path, writeErr)}
		}
		return configuration, nil
	}
	if err != nil {
		return configuration, []string{fmt.Sprintf(
			"configuration could not be read, using defaults: %v", err)}
	}

	var stored file
	if err := json.Unmarshal(contents, &stored); err != nil {
		return configuration, []string{fmt.Sprintf(
			"configuration is not valid JSON, using defaults: %v", err)}
	}

	// An unknown version is treated as a cache miss, not an error. This keeps a
	// downgrade working after a newer Brewtifyer has written the file.
	if stored.Version != nil && *stored.Version != Version {
		return configuration, []string{fmt.Sprintf(
			"configuration version %d is not supported, using defaults", *stored.Version)}
	}

	var notes []string
	if stored.CheckIntervalMinutes != nil {
		interval := time.Duration(*stored.CheckIntervalMinutes) * time.Minute
		clamped, note := clampDuration(
			"checkIntervalMinutes", interval, MinimumCheckInterval, MaximumCheckInterval)
		configuration.CheckInterval = clamped
		notes = appendNote(notes, note)
	}
	if stored.MaxVisibleUpdates != nil {
		clamped, note := clampInt(
			"maxVisibleUpdates", *stored.MaxVisibleUpdates,
			MinimumMaxVisibleUpdates, MaximumMaxVisibleUpdates)
		configuration.MaxVisibleUpdates = clamped
		notes = appendNote(notes, note)
	}
	if stored.TerminalApplication != nil && *stored.TerminalApplication != "" {
		configuration.TerminalApplication = *stored.TerminalApplication
	}
	if stored.BrewPath != nil {
		configuration.BrewPath = *stored.BrewPath
	}
	return configuration, notes
}

// Save writes the configuration atomically so that a crash or a full disk can
// never leave a half-written file behind.
func Save(path string, configuration Config) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("configuration directory could not be created: %w", err)
	}

	version := Version
	minutes := int(configuration.CheckInterval / time.Minute)
	maxVisible := configuration.MaxVisibleUpdates
	terminal := configuration.TerminalApplication
	brewPath := configuration.BrewPath
	contents, err := json.MarshalIndent(file{
		Version:              &version,
		CheckIntervalMinutes: &minutes,
		MaxVisibleUpdates:    &maxVisible,
		TerminalApplication:  &terminal,
		BrewPath:             &brewPath,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("configuration could not be encoded: %w", err)
	}
	contents = append(contents, '\n')

	temporary, err := os.CreateTemp(directory, ".config-*")
	if err != nil {
		return fmt.Errorf("temporary configuration could not be created: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := writeAndSync(temporary, contents); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("configuration could not be saved: %w", err)
	}
	removeTemporary = false
	return nil
}

func writeAndSync(target *os.File, contents []byte) error {
	if _, err := target.Write(contents); err != nil {
		_ = target.Close()
		return fmt.Errorf("configuration could not be written: %w", err)
	}
	if err := target.Sync(); err != nil {
		_ = target.Close()
		return fmt.Errorf("configuration could not be synchronized: %w", err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("configuration could not be closed: %w", err)
	}
	return nil
}

func clampDuration(name string, value, minimum, maximum time.Duration) (time.Duration, string) {
	switch {
	case value < minimum:
		return minimum, fmt.Sprintf("%s was raised to the minimum of %s", name, minimum)
	case value > maximum:
		return maximum, fmt.Sprintf("%s was lowered to the maximum of %s", name, maximum)
	default:
		return value, ""
	}
}

func clampInt(name string, value, minimum, maximum int) (int, string) {
	switch {
	case value < minimum:
		return minimum, fmt.Sprintf("%s was raised to the minimum of %d", name, minimum)
	case value > maximum:
		return maximum, fmt.Sprintf("%s was lowered to the maximum of %d", name, maximum)
	default:
		return value, ""
	}
}

func appendNote(notes []string, note string) []string {
	if note == "" {
		return notes
	}
	return append(notes, note)
}
