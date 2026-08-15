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
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
	"github.com/KevinCFechtel/Brewtifyer/internal/notification"
	trayui "github.com/KevinCFechtel/Brewtifyer/internal/tray"
	"github.com/KevinCFechtel/Brewtifyer/internal/upgrade"
)

const defaultCheckInterval = 6 * time.Hour

func main() {
	texts, err := localization.NewDetected()
	if err != nil {
		log.Fatalf("localization could not be initialized: %v", err)
	}

	brewPath, checker := newChecker(texts)
	resultHandler := newResultHandler(texts)
	var updater trayui.Updater
	if brewPath != "" {
		updater = upgrade.NewTerminalLauncher(brewPath, texts)
	}
	autostartController := autostart.NewNativeController()
	app := trayui.New(checker, defaultCheckInterval, resultHandler, updater, autostartController, texts)
	systray.Run(app.OnReady, app.OnExit)
}

func newResultHandler(texts *localization.Strings) func(brew.Result) {
	statePath, err := notification.DefaultStatePath()
	if err != nil {
		log.Printf("notifications were disabled: %v", err)
		return nil
	}

	service := notification.NewService(statePath, notification.NewNativeSender(), texts)
	return func(result brew.Result) {
		if err := service.Handle(result); err != nil {
			log.Printf("notification state could not be processed: %v", err)
		}
	}
}

func newChecker(texts *localization.Strings) (string, monitor.Checker) {
	brewPath, err := brew.Locate(os.Getenv("BREWTIFYER_BREW_PATH"))
	if err != nil {
		return "", monitor.CheckerFunc(func(context.Context) (brew.Result, error) {
			return brew.Result{}, errors.New(texts.HomebrewNotFound())
		})
	}

	return brewPath, brew.NewClient(brewPath)
}
