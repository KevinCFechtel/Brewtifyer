// Package upgrade opens interactive Homebrew upgrades in a terminal
// application, so that Homebrew stays visible and can prompt the user.
package upgrade

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
)

// openTimeout bounds the `open` call that hands the command file to the
// terminal application.
const (
	openTimeout             = 30 * time.Second
	completionPollInterval  = 500 * time.Millisecond
	completionWatchLifetime = 24 * time.Hour
)

// TerminalLauncher writes a short-lived .command file and opens it in a
// terminal application. Homebrew remains interactive and can ask for
// confirmation or credentials there.
type TerminalLauncher struct {
	configuredBrewPath string
	application        string
	tempDir            string
	resolveBrew        func(string) (string, error)
	openFile           func(string) error
	texts              *localization.Strings
	completed          chan struct{}
	closed             chan struct{}
	closeOnce          sync.Once
}

// NewTerminalLauncher opens upgrades in application, which is the name of a
// macOS app such as "Terminal", "iTerm" or "Ghostty". An empty name falls back
// to the system default so a bad configuration cannot disable upgrades.
func NewTerminalLauncher(configuredBrewPath, application string, texts *localization.Strings) *TerminalLauncher {
	if application == "" {
		application = config.DefaultTerminalApplication
	}
	launcher := &TerminalLauncher{
		configuredBrewPath: configuredBrewPath,
		application:        application,
		resolveBrew:        brew.Locate,
		texts:              texts,
		completed:          make(chan struct{}, 1),
		closed:             make(chan struct{}),
	}
	launcher.openFile = func(commandPath string) error {
		return openInTerminal(launcher.application, commandPath, texts)
	}
	return launcher
}

func (launcher *TerminalLauncher) UpgradePackage(currentPackage brew.Package) error {
	if currentPackage.Name == "" {
		return fmt.Errorf("%s", launcher.texts.PackageNameMissing())
	}
	if strings.ContainsRune(currentPackage.Name, '\x00') {
		return fmt.Errorf("%s", launcher.texts.PackageNameInvalid())
	}

	arguments := []string{"upgrade"}
	switch currentPackage.Kind {
	case brew.Formula:
		arguments = append(arguments, "--formula")
	case brew.Cask:
		arguments = append(arguments, "--cask")
	default:
		return fmt.Errorf("%s", launcher.texts.UnknownPackageKind(string(currentPackage.Kind)))
	}
	arguments = append(arguments, currentPackage.Name)
	return launcher.launch(arguments, launcher.texts.UpgradePackageDescription(currentPackage.Name), true)
}

func (launcher *TerminalLauncher) UpgradeKind(kind brew.Kind) error {
	switch kind {
	case brew.Formula:
		return launcher.launch(
			[]string{"upgrade", "--formula"},
			launcher.texts.UpgradeFormulaeDescription(),
			true,
		)
	case brew.Cask:
		return launcher.launch(
			[]string{"upgrade", "--cask"},
			launcher.texts.UpgradeCasksDescription(),
			true,
		)
	default:
		return fmt.Errorf("%s", launcher.texts.UnknownPackageKind(string(kind)))
	}
}

func (launcher *TerminalLauncher) UpgradeAll() error {
	return launcher.launch([]string{"upgrade"}, launcher.texts.UpgradeAllDescription(), true)
}

func (launcher *TerminalLauncher) ShowInfo(currentPackage brew.Package) error {
	if currentPackage.Name == "" {
		return fmt.Errorf("%s", launcher.texts.PackageNameMissing())
	}
	if strings.ContainsRune(currentPackage.Name, '\x00') {
		return fmt.Errorf("%s", launcher.texts.PackageNameInvalid())
	}

	arguments := []string{"info"}
	switch currentPackage.Kind {
	case brew.Formula:
		arguments = append(arguments, "--formula")
	case brew.Cask:
		arguments = append(arguments, "--cask")
	default:
		return fmt.Errorf("%s", launcher.texts.UnknownPackageKind(string(currentPackage.Kind)))
	}
	arguments = append(arguments, currentPackage.Name)
	return launcher.launch(arguments, launcher.texts.InfoPackageDescription(currentPackage.Name), false)
}

func (launcher *TerminalLauncher) Completed() <-chan struct{} { return launcher.completed }

func (launcher *TerminalLauncher) Close() {
	if launcher.closed == nil {
		return
	}
	launcher.closeOnce.Do(func() { close(launcher.closed) })
}

func (launcher *TerminalLauncher) launch(arguments []string, description string, notifyCompletion bool) error {
	brewPath, err := launcher.resolveBrew(launcher.configuredBrewPath)
	if err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.HomebrewNotFound(), err)
	}

	commandFile, err := os.CreateTemp(launcher.tempDir, "brewtifyer-command-*.command")
	if err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.CreateUpgradeCommandError(), err)
	}
	commandPath := commandFile.Name()
	completionPath := ""
	if notifyCompletion {
		completionFile, createErr := os.CreateTemp(launcher.tempDir, "brewtifyer-complete-*")
		if createErr != nil {
			_ = commandFile.Close()
			_ = os.Remove(commandPath)
			return fmt.Errorf("%s: %w", launcher.texts.CreateUpgradeCommandError(), createErr)
		}
		completionPath = completionFile.Name()
		_ = completionFile.Close()
		_ = os.Remove(completionPath)
	}

	keepCommand := false
	defer func() {
		_ = commandFile.Close()
		if !keepCommand {
			_ = os.Remove(commandPath)
			if completionPath != "" {
				_ = os.Remove(completionPath)
			}
		}
	}()

	if err := commandFile.Chmod(0o700); err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.MakeUpgradeCommandExecutableError(), err)
	}
	if _, err := commandFile.WriteString(commandScript(
		brewPath,
		arguments,
		description,
		completionPath,
		launcher.texts,
	)); err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.WriteUpgradeCommandError(), err)
	}
	if err := commandFile.Close(); err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.CloseUpgradeCommandError(), err)
	}

	if err := launcher.openFile(commandPath); err != nil {
		return err
	}
	keepCommand = true
	if completionPath != "" && launcher.completed != nil {
		go launcher.watchCompletion(completionPath)
	}
	return nil
}

func (launcher *TerminalLauncher) watchCompletion(path string) {
	ticker := time.NewTicker(completionPollInterval)
	defer ticker.Stop()
	timeout := time.NewTimer(completionWatchLifetime)
	defer timeout.Stop()
	defer func() { _ = os.Remove(path) }()

	for {
		select {
		case <-launcher.closed:
			return
		case <-timeout.C:
			return
		case <-ticker.C:
			if _, err := os.Stat(path); err == nil {
				select {
				case launcher.completed <- struct{}{}:
				default:
				}
				return
			} else if !os.IsNotExist(err) {
				return
			}
		}
	}
}

func openInTerminal(application, commandPath string, texts *localization.Strings) error {
	// `open` hands the file to the terminal app and returns immediately, so a
	// short budget is plenty and keeps a wedged launch from blocking the menu.
	ctx, cancel := context.WithTimeout(context.Background(), openTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, "/usr/bin/open", "-a", application, commandPath).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("%s: %w", texts.OpenTerminalError(), err)
		}
		return fmt.Errorf("%s: %s: %w", texts.OpenTerminalError(), message, err)
	}
	return nil
}

func commandScript(
	brewPath string,
	arguments []string,
	description string,
	completionPath string,
	texts *localization.Strings,
) string {
	command := make([]string, 0, len(arguments)+1)
	command = append(command, brewPath)
	command = append(command, arguments...)
	for index := range command {
		command[index] = shellQuote(command[index])
	}

	return "#!/bin/zsh\n" +
		"set -u\n" +
		"trap 'rm -f -- \"$0\"' EXIT\n\n" +
		"printf '\\e]0;Brewtifyer Update\\a'\n" +
		"printf '%s\\n\\n' " + shellQuote(description) + "\n" +
		strings.Join(command, " ") + "\n" +
		"update_status=$?\n" +
		completionSignalScript(completionPath) +
		"\nif (( update_status == 0 )); then\n" +
		"  printf '\\n%s\\n' " + shellQuote(texts.UpgradeCompleted()) + "\n" +
		"else\n" +
		"  printf " + shellQuote("\n"+texts.UpgradeFailedFormat()+"\n") + " \"$update_status\"\n" +
		"fi\n" +
		"printf '%s' " + shellQuote(texts.UpgradePressAnyKey()) + "\n" +
		"read -r -k 1\n" +
		"printf '\\n'\n" +
		"exit \"$update_status\"\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func completionSignalScript(path string) string {
	if path == "" {
		return ""
	}
	return "printf '%s\n' \"$update_status\" > " + shellQuote(path) + "\n"
}
