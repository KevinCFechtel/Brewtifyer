package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathIsUnderUserLogs(t *testing.T) {
	t.Parallel()

	path, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if filepath.Base(path) != logFileName {
		t.Errorf("Path() = %q, want it to end in %q", path, logFileName)
	}
	if !strings.Contains(path, filepath.Join("Library", "Logs", directory)) {
		t.Errorf("Path() = %q, want it under Library/Logs/%s", path, directory)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("Path() = %q, want an absolute path", path)
	}
}

func TestDescribePathAlwaysReturnsSomething(t *testing.T) {
	t.Parallel()

	if DescribePath() == "" {
		t.Fatal("DescribePath() is empty, want a usable hint for --help")
	}
}

func TestRotateMovesOversizedLog(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), logFileName)
	if err := os.WriteFile(path, make([]byte, maximumSize+1), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	rotate(path)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the oversized log is still in place, want it rotated away")
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("rotated log is missing: %v", err)
	}
}

func TestRotateKeepsSmallLog(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), logFileName)
	if err := os.WriteFile(path, []byte("still small\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	rotate(path)

	if _, err := os.Stat(path); err != nil {
		t.Errorf("a small log must not be rotated: %v", err)
	}
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Error("a small log must not produce a rotated generation")
	}
}

// A missing log is the normal first-run case and must not panic.
func TestRotateToleratesMissingLog(t *testing.T) {
	t.Parallel()

	rotate(filepath.Join(t.TempDir(), "does-not-exist.log"))
}
