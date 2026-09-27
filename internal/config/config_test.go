package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadWritesDefaultsWhenMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "config.json")
	configuration, notes := Load(path)

	if len(notes) != 0 {
		t.Fatalf("notes = %v, want none", notes)
	}
	if configuration != Default() {
		t.Fatalf("configuration = %#v, want %#v", configuration, Default())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default configuration was not written: %v", err)
	}

	// The written file must round-trip to the same settings.
	reloaded, notes := Load(path)
	if len(notes) != 0 {
		t.Fatalf("reload notes = %v, want none", notes)
	}
	if reloaded != configuration {
		t.Fatalf("reloaded = %#v, want %#v", reloaded, configuration)
	}
}

func TestLoadAppliesStoredValues(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `{
  "version": 1,
  "checkIntervalMinutes": 90,
  "maxVisibleUpdates": 25,
  "terminalApplication": "iTerm",
  "brewPath": "/custom/brew"
}`)

	configuration, notes := Load(path)
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want none", notes)
	}
	want := Config{
		CheckInterval:       90 * time.Minute,
		MaxVisibleUpdates:   25,
		TerminalApplication: "iTerm",
		BrewPath:            "/custom/brew",
	}
	if configuration != want {
		t.Fatalf("configuration = %#v, want %#v", configuration, want)
	}
}

func TestLoadFallsBackPerAbsentField(t *testing.T) {
	t.Parallel()

	// Only one setting is present; every other one must keep its default
	// instead of collapsing to the Go zero value.
	path := writeConfig(t, `{"version": 1, "maxVisibleUpdates": 3}`)

	configuration, notes := Load(path)
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want none", notes)
	}
	if configuration.MaxVisibleUpdates != 3 {
		t.Errorf("MaxVisibleUpdates = %d, want 3", configuration.MaxVisibleUpdates)
	}
	if configuration.CheckInterval != DefaultCheckInterval {
		t.Errorf("CheckInterval = %v, want %v", configuration.CheckInterval, DefaultCheckInterval)
	}
	if configuration.TerminalApplication != DefaultTerminalApplication {
		t.Errorf("TerminalApplication = %q, want %q",
			configuration.TerminalApplication, DefaultTerminalApplication)
	}
}

func TestLoadClampsOutOfRangeValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		contents          string
		wantInterval      time.Duration
		wantMaxVisible    int
		wantNoteSubstring string
	}{
		{
			name:              "interval below minimum",
			contents:          `{"version":1,"checkIntervalMinutes":1}`,
			wantInterval:      MinimumCheckInterval,
			wantMaxVisible:    DefaultMaxVisibleUpdates,
			wantNoteSubstring: "checkIntervalMinutes",
		},
		{
			name:              "interval above maximum",
			contents:          `{"version":1,"checkIntervalMinutes":100000}`,
			wantInterval:      MaximumCheckInterval,
			wantMaxVisible:    DefaultMaxVisibleUpdates,
			wantNoteSubstring: "checkIntervalMinutes",
		},
		{
			name:              "negative row count",
			contents:          `{"version":1,"maxVisibleUpdates":-5}`,
			wantInterval:      DefaultCheckInterval,
			wantMaxVisible:    MinimumMaxVisibleUpdates,
			wantNoteSubstring: "maxVisibleUpdates",
		},
		{
			name:              "row count above maximum",
			contents:          `{"version":1,"maxVisibleUpdates":9999}`,
			wantInterval:      DefaultCheckInterval,
			wantMaxVisible:    MaximumMaxVisibleUpdates,
			wantNoteSubstring: "maxVisibleUpdates",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			configuration, notes := Load(writeConfig(t, test.contents))
			if configuration.CheckInterval != test.wantInterval {
				t.Errorf("CheckInterval = %v, want %v", configuration.CheckInterval, test.wantInterval)
			}
			if configuration.MaxVisibleUpdates != test.wantMaxVisible {
				t.Errorf("MaxVisibleUpdates = %d, want %d",
					configuration.MaxVisibleUpdates, test.wantMaxVisible)
			}
			if len(notes) != 1 {
				t.Fatalf("notes = %v, want exactly one clamp note", notes)
			}
			if !strings.Contains(notes[0], test.wantNoteSubstring) {
				t.Errorf("note = %q, want it to mention %q", notes[0], test.wantNoteSubstring)
			}
		})
	}
}

// A newer Brewtifyer may write a version this build does not know. That must
// degrade to defaults instead of breaking, so downgrading stays possible.
func TestLoadFallsBackOnUnknownVersion(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `{"version": 99, "checkIntervalMinutes": 30}`)

	configuration, notes := Load(path)
	if configuration != Default() {
		t.Fatalf("configuration = %#v, want defaults", configuration)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %v, want one unsupported-version note", notes)
	}
}

func TestLoadFallsBackOnInvalidJSON(t *testing.T) {
	t.Parallel()

	configuration, notes := Load(writeConfig(t, `{"version": 1,`))
	if configuration != Default() {
		t.Fatalf("configuration = %#v, want defaults", configuration)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %v, want one parse note", notes)
	}
}

// Unknown fields belong to a future schema and must be ignored silently.
func TestLoadIgnoresUnknownFields(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `{"version":1,"maxVisibleUpdates":7,"futureSetting":{"a":1}}`)

	configuration, notes := Load(path)
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want none", notes)
	}
	if configuration.MaxVisibleUpdates != 7 {
		t.Fatalf("MaxVisibleUpdates = %d, want 7", configuration.MaxVisibleUpdates)
	}
}

// An empty terminal name would open no application at all, so it must not
// override the default.
func TestLoadIgnoresEmptyTerminalApplication(t *testing.T) {
	t.Parallel()

	configuration, _ := Load(writeConfig(t, `{"version":1,"terminalApplication":""}`))
	if configuration.TerminalApplication != DefaultTerminalApplication {
		t.Fatalf("TerminalApplication = %q, want %q",
			configuration.TerminalApplication, DefaultTerminalApplication)
	}
}

func TestSaveLeavesNoTemporaryFiles(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	if err := Save(path, Default()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("directory contains %v, want only config.json", names)
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
