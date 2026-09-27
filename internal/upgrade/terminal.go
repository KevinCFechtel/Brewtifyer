// Package upgrade opens interactive Homebrew upgrades in a terminal
// application, so that Homebrew stays visible and can prompt the user.
package upgrade

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
)

// openTimeout bounds the `open` call that hands the command file to the
// terminal application.
const openTimeout = 30 * time.Second

// TerminalLauncher writes a short-lived .command file and opens it in a
// terminal application. Homebrew remains interactive and can ask for
// confirmation or credentials there.
type TerminalLauncher struct {
	brewPath    string
	application string
	tempDir     string
	openFile    func(string) error
	texts       *localization.Strings
}

// NewTerminalLauncher opens upgrades in application, which is the name of a
// macOS app such as "Terminal", "iTerm" or "Ghostty". An empty name falls back
// to the system default so a bad configuration cannot disable upgrades.
func NewTerminalLauncher(brewPath, application string, texts *localization.Strings) *TerminalLauncher {
	if application == "" {
		application = config.DefaultTerminalApplication
	}
	launcher := &TerminalLauncher{
		brewPath:    brewPath,
		application: application,
		texts:       texts,
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
	return launcher.launch(arguments, launcher.texts.UpgradePackageDescription(currentPackage.Name))
}

func (launcher *TerminalLauncher) UpgradeAll() error {
	return launcher.launch([]string{"upgrade"}, launcher.texts.UpgradeAllDescription())
}

func (launcher *TerminalLauncher) launch(arguments []string, description string) error {
	commandFile, err := os.CreateTemp(launcher.tempDir, "brewtifyer-upgrade-*.command")
	if err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.CreateUpgradeCommandError(), err)
	}
	commandPath := commandFile.Name()
	keepCommand := false
	defer func() {
		_ = commandFile.Close()
		if !keepCommand {
			_ = os.Remove(commandPath)
		}
	}()

	if err := commandFile.Chmod(0o700); err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.MakeUpgradeCommandExecutableError(), err)
	}
	if _, err := commandFile.WriteString(commandScript(launcher.brewPath, arguments, description, launcher.texts)); err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.WriteUpgradeCommandError(), err)
	}
	if err := commandFile.Close(); err != nil {
		return fmt.Errorf("%s: %w", launcher.texts.CloseUpgradeCommandError(), err)
	}

	if err := launcher.openFile(commandPath); err != nil {
		return err
	}
	keepCommand = true
	return nil
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

func commandScript(brewPath string, arguments []string, description string, texts *localization.Strings) string {
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
		"update_status=$?\n\n" +
		"if (( update_status == 0 )); then\n" +
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
