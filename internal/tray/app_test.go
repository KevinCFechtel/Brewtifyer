package tray

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
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

	icon, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode icon: %v", err)
	}
	_, _, _, cornerAlpha := icon.At(icon.Bounds().Min.X, icon.Bounds().Min.Y).RGBA()
	if cornerAlpha != 0 {
		t.Fatalf("corner alpha = %d, want transparent menu bar background", cornerAlpha)
	}

	hasVisiblePixel := false
	for y := icon.Bounds().Min.Y; y < icon.Bounds().Max.Y && !hasVisiblePixel; y++ {
		for x := icon.Bounds().Min.X; x < icon.Bounds().Max.X; x++ {
			_, _, _, alpha := icon.At(x, y).RGBA()
			if alpha != 0 {
				hasVisiblePixel = true
				break
			}
		}
	}
	if !hasVisiblePixel {
		t.Fatal("menu bar icon contains no visible pixels")
	}
}

func TestAutostartMenuState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		status       autostart.Status
		checked      bool
		enabled      bool
		showSettings bool
	}{
		{name: "unsupported", status: autostart.Unsupported},
		{name: "disabled", status: autostart.Disabled, enabled: true},
		{name: "enabled", status: autostart.Enabled, checked: true, enabled: true},
		{
			name:         "requires approval",
			status:       autostart.RequiresApproval,
			enabled:      true,
			showSettings: true,
		},
		{name: "not found", status: autostart.NotFound, enabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			state := autostartMenuStateFor(test.status)
			if state.checked != test.checked || state.enabled != test.enabled || state.showSettings != test.showSettings {
				t.Fatalf(
					"state = {checked:%t enabled:%t showSettings:%t}",
					state.checked,
					state.enabled,
					state.showSettings,
				)
			}
		})
	}
}

func TestAutostartToggle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status        autostart.Status
		wantEnabled   bool
		wantCanToggle bool
	}{
		{status: autostart.Disabled, wantEnabled: true, wantCanToggle: true},
		{status: autostart.Enabled, wantEnabled: false, wantCanToggle: true},
		{status: autostart.RequiresApproval},
		{status: autostart.Unsupported},
		{status: autostart.NotFound, wantEnabled: true, wantCanToggle: true},
	}

	for _, test := range tests {
		enabled, canToggle := autostartToggle(test.status)
		if enabled != test.wantEnabled || canToggle != test.wantCanToggle {
			t.Fatalf(
				"autostartToggle(%d) = (%t, %t), want (%t, %t)",
				test.status,
				enabled,
				canToggle,
				test.wantEnabled,
				test.wantCanToggle,
			)
		}
	}
}
