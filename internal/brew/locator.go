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
			return "", fmt.Errorf("configured Homebrew path is invalid: %w", err)
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

	return "", errors.New("not found in PATH or at a standard location")
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
		return "", errors.New("path points to a directory")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", errors.New("file is not executable")
	}

	return absolutePath, nil
}
