package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
)

const terminalApplication = "Terminal"

// TerminalLauncher writes a short-lived .command file and opens it in the
// macOS Terminal app. Homebrew remains interactive and can ask for confirmation
// or credentials there.
type TerminalLauncher struct {
	brewPath string
	tempDir  string
	openFile func(string) error
}

func NewTerminalLauncher(brewPath string) *TerminalLauncher {
	return &TerminalLauncher{
		brewPath: brewPath,
		openFile: openInTerminal,
	}
}

func (launcher *TerminalLauncher) UpgradePackage(currentPackage brew.Package) error {
	if currentPackage.Name == "" {
		return fmt.Errorf("Paketname fehlt")
	}
	if strings.ContainsRune(currentPackage.Name, '\x00') {
		return fmt.Errorf("Paketname enthält ein ungültiges Zeichen")
	}

	arguments := []string{"upgrade"}
	switch currentPackage.Kind {
	case brew.Formula:
		arguments = append(arguments, "--formula")
	case brew.Cask:
		arguments = append(arguments, "--cask")
	default:
		return fmt.Errorf("unbekannter Pakettyp: %q", currentPackage.Kind)
	}
	arguments = append(arguments, currentPackage.Name)
	return launcher.launch(arguments, "Homebrew-Update für "+currentPackage.Name)
}

func (launcher *TerminalLauncher) UpgradeAll() error {
	return launcher.launch([]string{"upgrade"}, "Alle Homebrew-Updates")
}

func (launcher *TerminalLauncher) launch(arguments []string, description string) error {
	commandFile, err := os.CreateTemp(launcher.tempDir, "brewtifyer-upgrade-*.command")
	if err != nil {
		return fmt.Errorf("temporärer Update-Befehl konnte nicht erstellt werden: %w", err)
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
		return fmt.Errorf("Update-Befehl konnte nicht ausführbar gemacht werden: %w", err)
	}
	if _, err := commandFile.WriteString(commandScript(launcher.brewPath, arguments, description)); err != nil {
		return fmt.Errorf("Update-Befehl konnte nicht geschrieben werden: %w", err)
	}
	if err := commandFile.Close(); err != nil {
		return fmt.Errorf("Update-Befehl konnte nicht geschlossen werden: %w", err)
	}

	if err := launcher.openFile(commandPath); err != nil {
		return err
	}
	keepCommand = true
	return nil
}

func openInTerminal(commandPath string) error {
	output, err := exec.Command("/usr/bin/open", "-a", terminalApplication, commandPath).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("Terminal konnte nicht geöffnet werden: %w", err)
		}
		return fmt.Errorf("Terminal konnte nicht geöffnet werden: %s: %w", message, err)
	}
	return nil
}

func commandScript(brewPath string, arguments []string, description string) string {
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
		"  printf '\\nUpdate abgeschlossen. Prüfe Brewtifyer anschließend erneut.\\n'\n" +
		"else\n" +
		"  printf '\\nUpdate mit Status %d beendet.\\n' \"$update_status\"\n" +
		"fi\n" +
		"printf 'Zum Schließen des Fensters eine Taste drücken …'\n" +
		"read -r -k 1\n" +
		"printf '\\n'\n" +
		"exit \"$update_status\"\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
