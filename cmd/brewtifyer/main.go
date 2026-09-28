package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/KevinCFechtel/Brewtifyer/internal/autostart"
	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/buildinfo"
	"github.com/KevinCFechtel/Brewtifyer/internal/config"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
	"github.com/KevinCFechtel/Brewtifyer/internal/logging"
	"github.com/KevinCFechtel/Brewtifyer/internal/monitor"
	"github.com/KevinCFechtel/Brewtifyer/internal/notification"
	trayui "github.com/KevinCFechtel/Brewtifyer/internal/tray"
	"github.com/KevinCFechtel/Brewtifyer/internal/upgrade"
)

// brewVersionTimeout bounds the diagnostic version lookup at startup so that a
// hanging Homebrew cannot delay the menu bar icon from appearing.
const brewVersionTimeout = 20 * time.Second

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("Brewtifyer %s\n", buildinfo.Summary())
		return
	}
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument: %s\n\n", flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}

	os.Exit(run())
}

// run holds the application body so that deferred cleanup, in particular
// closing the log file, still happens on every exit path. Calling log.Fatal
// from here would skip those defers.
func run() int {
	closeLog := logging.Setup()
	defer closeLog()

	log.Printf("starting Brewtifyer %s", buildinfo.Summary())

	configuration := loadConfiguration()

	texts, err := localization.NewDetected()
	if err != nil {
		log.Printf("localization could not be initialized: %v", err)
		return 1
	}
	log.Printf("language: %s", texts.Language())

	configuredPath := configuredBrewPath(configuration)
	checker := newChecker(configuredPath)
	updater := upgrade.NewTerminalLauncher(configuredPath, configuration.TerminalApplication, texts)

	app := trayui.New(trayui.Options{
		Menu:          trayui.SystrayMenu(),
		Checker:       checker,
		Config:        configuration,
		ResultHandler: newResultHandler(texts),
		Updater:       updater,
		Autostart:     autostart.NewNativeController(),
		Texts:         texts,
	})
	trayui.Run(app)
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `Brewtifyer %s — a macOS menu bar app for Homebrew updates.

Usage:
  Brewtifyer [flags]

Flags:
  -version    print version information and exit
  -help       print this message

Environment:
  BREWTIFYER_BREW_PATH   path to the brew executable, overriding autodetection
  BREWTIFYER_LANGUAGE    force the interface language ("en" or "de")

Configuration and logs:
  ~/Library/Application Support/Brewtifyer/config.json
  %s
`, buildinfo.Summary(), logging.DescribePath())
}

// loadConfiguration never fails: a missing, broken, or future config file
// degrades to the defaults so that the app always starts.
func loadConfiguration() config.Config {
	path, err := config.DefaultPath()
	if err != nil {
		log.Printf("configuration path could not be determined, using defaults: %v", err)
		return config.Default()
	}

	configuration, notes := config.Load(path)
	for _, note := range notes {
		log.Printf("configuration: %s", note)
	}
	log.Printf("configuration: interval %s, %d rows, terminal %q (%s)",
		configuration.CheckInterval,
		configuration.MaxVisibleUpdates,
		configuration.TerminalApplication,
		path,
	)
	return configuration
}

func newResultHandler(texts *localization.Strings) func(brew.Result) {
	statePath, err := notification.DefaultStatePath()
	if err != nil {
		log.Printf("notifications were disabled: %v", err)
		return nil
	}

	service := notification.NewService(statePath, notification.NewNativeSender(), texts)
	return func(result brew.Result) {
		note, err := service.Handle(result)
		if note != "" {
			log.Printf("notification state: %s", note)
		}
		if err != nil {
			log.Printf("notification state could not be processed: %v", err)
		}
	}
}

func configuredBrewPath(configuration config.Config) string {
	if fromEnvironment := os.Getenv("BREWTIFYER_BREW_PATH"); fromEnvironment != "" {
		// The environment variable wins so that a single run can be redirected
		// without editing the configuration file.
		return fromEnvironment
	}
	return configuration.BrewPath
}

// newChecker resolves Homebrew for every check. The lookup is cheap compared
// with invoking brew and lets a long-running app recover when Homebrew is
// installed, moved, or replaced without requiring a restart.
func newChecker(configuredPath string) monitor.Checker {
	var lastPath string

	if brewPath, err := brew.Locate(configuredPath); err != nil {
		log.Printf("Homebrew could not be located at startup: %v", err)
	} else {
		lastPath = brewPath
		log.Printf("Homebrew executable: %s", brewPath)
		logBrewVersion(brew.NewClient(brewPath))
	}

	return monitor.CheckerFunc(func(ctx context.Context) (brew.Result, error) {
		brewPath, err := brew.Locate(configuredPath)
		if err != nil {
			return brew.Result{}, err
		}
		if brewPath != lastPath {
			log.Printf("Homebrew executable changed: %s", brewPath)
			lastPath = brewPath
		}
		return brew.NewClient(brewPath).Check(ctx)
	})
}

// logBrewVersion records the Homebrew version for bug reports. Brewtifyer never
// gates behaviour on it: an incompatible output format is detected from the
// output itself, so a failure here is only worth a log line.
func logBrewVersion(client *brew.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), brewVersionTimeout)
	defer cancel()

	version, err := client.Version(ctx)
	if err != nil {
		log.Printf("Homebrew version could not be determined: %v", err)
		return
	}
	log.Printf("Homebrew version: %s", version)
}
