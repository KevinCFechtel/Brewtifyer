package upgrade

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
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
		brewPath: "/opt/homebrew/bin/brew",
		tempDir:  temporaryDirectory,
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
		brewPath: "/opt/homebrew/bin/brew",
		tempDir:  temporaryDirectory,
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

	launcher := NewTerminalLauncher("/opt/homebrew/bin/brew")
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
	)
	command := exec.Command(zshPath, "-n")
	command.Stdin = strings.NewReader(script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("zsh rejected generated command: %v\n%s", err, output)
	}
}

func testLauncher(t *testing.T) (*TerminalLauncher, <-chan string) {
	t.Helper()

	openedScript := make(chan string, 1)
	launcher := &TerminalLauncher{
		brewPath: "/opt/homebrew/bin/brew",
		tempDir:  filepath.Clean(t.TempDir()),
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
