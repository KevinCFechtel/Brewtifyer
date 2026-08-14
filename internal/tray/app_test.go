package tray

import (
	"testing"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
)

func TestPackageTitle(t *testing.T) {
	t.Parallel()

	title := packageTitle(brew.Package{
		Name:              "go",
		InstalledVersions: []string{"1.26.5"},
		CurrentVersion:    "1.26.6",
		Pinned:            true,
	})
	if title != "go: 1.26.5 → 1.26.6 · angeheftet" {
		t.Fatalf("packageTitle() = %q", title)
	}
}

func TestIconIsPNG(t *testing.T) {
	t.Parallel()

	data := iconPNG()
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("iconPNG() did not return a PNG image")
	}
}
