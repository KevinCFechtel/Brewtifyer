package upgrade

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
)

func TestUpgradePackageCreatesFormulaCommand(t *testing.T) {
	t.Parallel()

	launcher, openedScript := testLauncher(t)
	err := launcher.UpgradePackage(brew.Package{
		Name: "go@1.26",
		Kind: brew.Formula,
	})
	if err != nil {
		t.Fatalf("UpgradePackage() error = %v", err)
	}

	script := <-openedScript
	want := "'/opt/homebrew/bin/brew' 'upgrade' '--formula' 'go@1.26'"
	if !strings.Contains(script, want) {
		t.Fatalf("script does not contain %q:\n%s", want, script)
	}
}

func TestUpgradePackageCreatesCaskCommand(t *testing.T) {
	t.Parallel()

	launcher, openedScript := testLauncher(t)
	err := launcher.UpgradePackage(brew.Package{
		Name: "firefox",
		Kind: brew.Cask,
	})
	if err != nil {
		t.Fatalf("UpgradePackage() error = %v", err)
	}

	script := <-openedScript
	want := "'/opt/homebrew/bin/brew' 'upgrade' '--cask' 'firefox'"
	if !strings.Contains(script, want) {
		t.Fatalf("script does not contain %q:\n%s", want, script)
	}
}

func TestUpgradeAllUsesPlainUpgradeCommand(t *testing.T) {
	t.Parallel()

	launcher, openedScript := testLauncher(t)
	if err := launcher.UpgradeAll(); err != nil {
		t.Fatalf("UpgradeAll() error = %v", err)
	}

	script := <-openedScript
	if !strings.Contains(script, "'/opt/homebrew/bin/brew' 'upgrade'\n") {
		t.Fatalf("script does not contain plain upgrade command:\n%s", script)
	}
	if strings.Contains(script, "--formula") || strings.Contains(script, "--cask") {
		t.Fatalf("all-packages script unexpectedly restricts the package kind:\n%s", script)
	}
}

func TestUpgradeKindCreatesScopedCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind brew.Kind
		want string
	}{
		{kind: brew.Formula, want: "'/opt/homebrew/bin/brew' 'upgrade' '--formula'"},
		{kind: brew.Cask, want: "'/opt/homebrew/bin/brew' 'upgrade' '--cask'"},
	}
	for _, test := range tests {
		launcher, openedScript := testLauncher(t)
		if err := launcher.UpgradeKind(test.kind); err != nil {
			t.Fatalf("UpgradeKind(%q) error = %v", test.kind, err)
		}
		if script := <-openedScript; !strings.Contains(script, test.want) {
			t.Fatalf("script does not contain %q:\n%s", test.want, script)
		}
	}
}

func TestShowInfoUsesPackageKind(t *testing.T) {
	t.Parallel()

	launcher, openedScript := testLauncher(t)
	if err := launcher.ShowInfo(brew.Package{Name: "firefox", Kind: brew.Cask}); err != nil {
		t.Fatalf("ShowInfo() error = %v", err)
	}
	script := <-openedScript
	if !strings.Contains(script, "'/opt/homebrew/bin/brew' 'info' '--cask' 'firefox'") {
		t.Fatalf("info script uses unexpected command:\n%s", script)
	}
	if strings.Contains(script, "brewtifyer-complete-") {
		t.Fatalf("info command unexpectedly contains upgrade completion marker:\n%s", script)
	}
	if strings.Contains(script, "Update completed") {
		t.Fatalf("info command unexpectedly contains upgrade completion messaging:\n%s", script)
	}
}

func TestPackageNameIsShellQuoted(t *testing.T) {
	t.Parallel()

	launcher, openedScript := testLauncher(t)
	err := launcher.UpgradePackage(brew.Package{
		Name: "example'; echo unsafe; '",
		Kind: brew.Formula,
	})
	if err != nil {
		t.Fatalf("UpgradePackage() error = %v", err)
	}

	script := <-openedScript
	want := "'example'\"'\"'; echo unsafe; '\"'\"''"
	if !strings.Contains(script, want) {
		t.Fatalf("package name was not safely quoted:\n%s", script)
	}
}

func TestCommandFileIsExecutableAndSelfRemoving(t *testing.T) {
	t.Parallel()

	temporaryDirectory := t.TempDir()
	launcher := &TerminalLauncher{
		configuredBrewPath: "/configured/brew",
		tempDir:            temporaryDirectory,
		resolveBrew:        func(string) (string, error) { return "/opt/homebrew/bin/brew", nil },
		texts:              localization.MustNew("de"),
	}
	launcher.openFile = func(commandPath string) error {
		information, err := os.Stat(commandPath)
		if err != nil {
			return err
		}
		if permissions := information.Mode().Perm(); permissions != 0o700 {
			t.Fatalf("command permissions = %o, want 700", permissions)
		}
		content, err := os.ReadFile(commandPath)
		if err != nil {
			return err
		}
		if !strings.Contains(string(content), `trap 'rm -f -- "$0"' EXIT`) {
			t.Fatal("command does not remove itself on exit")
		}
		return nil
	}

	if err := launcher.UpgradeAll(); err != nil {
		t.Fatalf("UpgradeAll() error = %v", err)
	}
}

func TestFailedOpenRemovesCommandFile(t *testing.T) {
	t.Parallel()

	temporaryDirectory := t.TempDir()
	launcher := &TerminalLauncher{
		configuredBrewPath: "/configured/brew",
		tempDir:            temporaryDirectory,
		resolveBrew:        func(string) (string, error) { return "/opt/homebrew/bin/brew", nil },
		texts:              localization.MustNew("de"),
		openFile: func(string) error {
			return errors.New("open failed")
		},
	}

	if err := launcher.UpgradeAll(); err == nil {
		t.Fatal("UpgradeAll() error = nil, want open error")
	}
	entries, err := os.ReadDir(temporaryDirectory)
	if err != nil {
		t.Fatalf("read temp directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary directory contains %d files, want none", len(entries))
	}
}

func TestUpgradePackageRejectsUnknownKind(t *testing.T) {
	t.Parallel()

	launcher := NewTerminalLauncher("/opt/homebrew/bin/brew", "Terminal", localization.MustNew("de"))
	err := launcher.UpgradePackage(brew.Package{Name: "example", Kind: "unknown"})
	if err == nil {
		t.Fatal("UpgradePackage() error = nil, want unknown kind error")
	}
}

func TestGeneratedCommandHasValidZshSyntax(t *testing.T) {
	t.Parallel()

	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not available")
	}
	script := commandScript(
		"/opt/homebrew/bin/brew",
		[]string{"upgrade", "--formula", "example'; echo unsafe; '"},
		"Homebrew-Update für example'; echo unsafe; '",
		"/tmp/brewtifyer-test-complete",
		localization.MustNew("de"),
	)
	command := exec.CommandContext(t.Context(), zshPath, "-n")
	command.Stdin = strings.NewReader(script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("zsh rejected generated command: %v\n%s", err, output)
	}
}

func TestGeneratedCommandUsesSelectedLanguage(t *testing.T) {
	t.Parallel()

	texts := localization.MustNew("en")
	script := commandScript(
		"/opt/homebrew/bin/brew",
		[]string{"upgrade"},
		texts.UpgradeAllDescription(),
		"/tmp/brewtifyer-test-complete",
		texts,
	)
	for _, expected := range []string{
		"All Homebrew updates",
		"Update completed. Brewtifyer will check again automatically.",
		"Press any key to close the window …",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script does not contain %q:\n%s", expected, script)
		}
	}
}

func testLauncher(t *testing.T) (*TerminalLauncher, <-chan string) {
	t.Helper()

	openedScript := make(chan string, 1)
	launcher := &TerminalLauncher{
		configuredBrewPath: "/configured/brew",
		tempDir:            filepath.Clean(t.TempDir()),
		resolveBrew:        func(string) (string, error) { return "/opt/homebrew/bin/brew", nil },
		texts:              localization.MustNew("de"),
	}
	launcher.openFile = func(commandPath string) error {
		content, err := os.ReadFile(commandPath)
		if err != nil {
			return err
		}
		openedScript <- string(content)
		return nil
	}
	return launcher, openedScript
}

func TestLauncherResolvesHomebrewForEachUpgrade(t *testing.T) {
	t.Parallel()

	launcher, openedScript := testLauncher(t)
	calls := 0
	launcher.resolveBrew = func(configured string) (string, error) {
		calls++
		if configured != "/configured/brew" {
			t.Fatalf("configured path = %q, want %q", configured, "/configured/brew")
		}
		if calls == 1 {
			return "/first/brew", nil
		}
		return "/second/brew", nil
	}

	if err := launcher.UpgradeAll(); err != nil {
		t.Fatalf("first UpgradeAll() error = %v", err)
	}
	if script := <-openedScript; !strings.Contains(script, "'/first/brew' 'upgrade'") {
		t.Fatalf("first script used unexpected brew path:\n%s", script)
	}
	if err := launcher.UpgradeAll(); err != nil {
		t.Fatalf("second UpgradeAll() error = %v", err)
	}
	if script := <-openedScript; !strings.Contains(script, "'/second/brew' 'upgrade'") {
		t.Fatalf("second script used unexpected brew path:\n%s", script)
	}
	if calls != 2 {
		t.Fatalf("resolve calls = %d, want 2", calls)
	}
}

func TestNewTerminalLauncherFallsBackToDefaultApplication(t *testing.T) {
	t.Parallel()

	launcher := NewTerminalLauncher("/opt/homebrew/bin/brew", "", localization.MustNew("en"))
	if launcher.application != config.DefaultTerminalApplication {
		t.Fatalf("application = %q, want %q",
			launcher.application, config.DefaultTerminalApplication)
	}
}

func TestNewTerminalLauncherKeepsConfiguredApplication(t *testing.T) {
	t.Parallel()

	launcher := NewTerminalLauncher("/opt/homebrew/bin/brew", "Ghostty", localization.MustNew("en"))
	if launcher.application != "Ghostty" {
		t.Fatalf("application = %q, want %q", launcher.application, "Ghostty")
	}
}
