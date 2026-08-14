package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"fyne.io/systray"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
	"github.com/KevinCFechtel/Brewtifyer/internal/notification"
	trayui "github.com/KevinCFechtel/Brewtifyer/internal/tray"
	"github.com/KevinCFechtel/Brewtifyer/internal/upgrade"
)

const defaultCheckInterval = 6 * time.Hour

func main() {
	brewPath, checker := newChecker()
	resultHandler := newResultHandler()
	var updater trayui.Updater
	if brewPath != "" {
		updater = upgrade.NewTerminalLauncher(brewPath)
	}
	autostartController := autostart.NewNativeController()
	app := trayui.New(checker, defaultCheckInterval, resultHandler, updater, autostartController)
	systray.Run(app.OnReady, app.OnExit)
}

func newResultHandler() func(brew.Result) {
	statePath, err := notification.DefaultStatePath()
	if err != nil {
		log.Printf("Benachrichtigungen wurden deaktiviert: %v", err)
		return nil
	}

	service := notification.NewService(statePath, notification.NewNativeSender())
	return func(result brew.Result) {
		if err := service.Handle(result); err != nil {
			log.Printf("Benachrichtigungszustand konnte nicht verarbeitet werden: %v", err)
		}
	}
}

func newChecker() (string, monitor.Checker) {
	brewPath, err := brew.Locate(os.Getenv("BREWTIFYER_BREW_PATH"))
	if err != nil {
		return "", monitor.CheckerFunc(func(context.Context) (brew.Result, error) {
			return brew.Result{}, errors.New("Homebrew wurde nicht gefunden")
		})
	}

	return brewPath, brew.NewClient(brewPath)
}
