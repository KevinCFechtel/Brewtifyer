package main

import (
	"context"
	"errors"
	"os"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
	trayui "github.com/KevinCFechtel/Brewtifyer/internal/tray"
)

const defaultCheckInterval = 6 * time.Hour

func main() {
	checker := newChecker()
	app := trayui.New(checker, defaultCheckInterval)
	systray.Run(app.OnReady, app.OnExit)
}

func newChecker() monitor.Checker {
	brewPath, err := brew.Locate(os.Getenv("BREWTIFYER_BREW_PATH"))
	if err != nil {
		return monitor.CheckerFunc(func(context.Context) (brew.Result, error) {
			return brew.Result{}, errors.New("Homebrew wurde nicht gefunden")
		})
	}

	return brew.NewClient(brewPath)
}
