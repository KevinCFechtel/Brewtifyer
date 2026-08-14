package brew

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var defaultBrewPaths = []string{
	"/opt/homebrew/bin/brew",
	"/usr/local/bin/brew",
}

// Locate finds a usable Homebrew executable. An explicitly configured path
// takes precedence because GUI applications often receive a reduced PATH.
func Locate(configuredPath string) (string, error) {
	if configuredPath != "" {
		path, err := validateExecutable(configuredPath)
		if err != nil {
			return "", fmt.Errorf("konfigurierter Homebrew-Pfad ist ungültig: %w", err)
		}
		return path, nil
	}

	if path, err := exec.LookPath("brew"); err == nil {
		if validated, validateErr := validateExecutable(path); validateErr == nil {
			return validated, nil
		}
	}

	for _, path := range defaultBrewPaths {
		if validated, err := validateExecutable(path); err == nil {
			return validated, nil
		}
	}

	return "", errors.New("weder im PATH noch an einem Standardpfad gefunden")
}

func validateExecutable(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("Pfad bezeichnet ein Verzeichnis")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("Datei ist nicht ausführbar")
	}

	return absolutePath, nil
}
