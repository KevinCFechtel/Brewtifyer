package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
)

const terminalApplication = "Terminal"

// TerminalLauncher writes a short-lived .command file and opens it in the
// macOS Terminal app. Homebrew remains interactive and can ask for confirmation
// or credentials there.
type TerminalLauncher struct {
	brewPath string
	tempDir  string
	openFile func(string) error
	texts    *localization.Strings
}

func NewTerminalLauncher(brewPath string, texts *localization.Strings) *TerminalLauncher {
	launcher := &TerminalLauncher{
		brewPath: brewPath,
		texts:    texts,
	}
	launcher.openFile = func(commandPath string) error {
		return openInTerminal(commandPath, texts)
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

func openInTerminal(commandPath string, texts *localization.Strings) error {
	output, err := exec.Command("/usr/bin/open", "-a", terminalApplication, commandPath).CombinedOutput()
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
