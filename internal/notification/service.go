// Package notification sends native macOS update notifications and remembers
// which package versions were already announced, so a restart does not repeat
// them. The remembered state is a cache and is rebuilt whenever it cannot be
// interpreted.
package notification

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
)

const stateVersion = 1

// Sender delivers a notification through the platform notification center.
type Sender interface {
	Send(title, body string)
}

// Service remembers the update set from the last successful check and only
// attempts to notify about package versions that were not present in that set.
// The persisted state means "seen by Brewtifyer", not "confirmed delivered":
// native notification authorization and delivery complete asynchronously.
type Service struct {
	statePath string
	sender    Sender
	texts     *localization.Strings
	mutex     sync.Mutex
}

type state struct {
	Version  int            `json:"version"`
	Packages []packageState `json:"packages"`
}

type packageState struct {
	Name    string    `json:"name"`
	Kind    brew.Kind `json:"kind"`
	Version string    `json:"version"`
}

func NewService(statePath string, sender Sender, texts *localization.Strings) *Service {
	return &Service{
		statePath: statePath,
		sender:    sender,
		texts:     texts,
	}
}

func DefaultStatePath() (string, error) {
	configurationDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user configuration directory could not be determined: %w", err)
	}
	return filepath.Join(configurationDirectory, "Brewtifyer", "notification-state.json"), nil
}

// Handle compares the result against the remembered state and attempts a
// notification for package versions that were not present before. State is
// saved before delivery because the native sender is asynchronous; this avoids
// duplicate notifications across restarts even when macOS suppresses one.
 // file is reported through the returned note; it is not an error, because the
// file is reported through the returned note; it is not an error, because the
// only consequence is a repeated notification.
func (service *Service) Handle(result brew.Result) (note string, err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	previous, note := loadState(service.statePath)

	current := packageStates(result.Packages)
	if equalPackageStates(previous.Packages, current) {
		return note, nil
	}

	newPackages := newlyAvailable(result.Packages, previous.Packages)
	if err := saveState(service.statePath, state{
		Version:  stateVersion,
		Packages: current,
	}); err != nil {
		return note, err
	}

	if len(newPackages) > 0 && service.sender != nil {
		title, body := message(service.texts, newPackages)
		service.sender.Send(title, body)
	}
	return note, nil
}

// loadState reads the deduplication state. The state is a cache, not a source of
// truth: anything that cannot be interpreted is discarded and reported through
// note rather than returned as an error. Returning an error here would be worse
// than starting over, because Handle would then never reach saveState and
// notifications would stay broken until the file was deleted by hand. The only
// cost of discarding is one repeated notification.
func loadState(statePath string) (loaded state, note string) {
	fresh := state{Version: stateVersion}

	file, err := os.Open(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return fresh, ""
	}
	if err != nil {
		return fresh, fmt.Sprintf("notification state could not be opened, starting over: %v", err)
	}
	// Read-only handle: a Close error cannot affect the result.
	defer func() { _ = file.Close() }()

	var saved state
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&saved); err != nil {
		return fresh, fmt.Sprintf("notification state could not be read, starting over: %v", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return fresh, fmt.Sprintf("notification state was incomplete, starting over: %v", err)
	}
	// An unknown version belongs to a newer Brewtifyer. Treat it as a cache miss
	// so that downgrading keeps working.
	if saved.Version != stateVersion {
		return fresh, fmt.Sprintf(
			"notification state version %d is not supported, starting over", saved.Version)
	}
	sortPackageStates(saved.Packages)
	return saved, ""
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("notification state could not be read completely: %w", err)
	}
	return errors.New("notification state contains additional data")
}

func saveState(statePath string, current state) error {
	directory := filepath.Dir(statePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("notification state directory could not be created: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".notification-state-*")
	if err != nil {
		return fmt.Errorf("temporary notification state could not be created: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(current); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("notification state could not be written: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("notification state could not be synchronized: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("notification state could not be closed: %w", err)
	}
	if err := os.Rename(temporaryPath, statePath); err != nil {
		return fmt.Errorf("notification state could not be saved: %w", err)
	}
	removeTemporary = false
	return nil
}

func packageStates(packages []brew.Package) []packageState {
	unique := make(map[string]packageState, len(packages))
	for _, currentPackage := range packages {
		recorded := packageState{
			Name:    currentPackage.Name,
			Kind:    currentPackage.Kind,
			Version: currentPackage.CurrentVersion,
		}
		unique[recorded.key()] = recorded
	}

	states := make([]packageState, 0, len(unique))
	for _, recorded := range unique {
		states = append(states, recorded)
	}
	sortPackageStates(states)
	return states
}

func newlyAvailable(packages []brew.Package, previous []packageState) []brew.Package {
	known := make(map[string]struct{}, len(previous))
	for _, recorded := range previous {
		known[recorded.key()] = struct{}{}
	}

	seen := make(map[string]struct{}, len(packages))
	var newlyAvailable []brew.Package
	for _, currentPackage := range packages {
		key := packageState{
			Name:    currentPackage.Name,
			Kind:    currentPackage.Kind,
			Version: currentPackage.CurrentVersion,
		}.key()
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		if _, exists := known[key]; !exists {
			newlyAvailable = append(newlyAvailable, currentPackage)
		}
	}
	return newlyAvailable
}

func equalPackageStates(left, right []packageState) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sortPackageStates(states []packageState) {
	slices.SortFunc(states, func(left, right packageState) int {
		return strings.Compare(left.key(), right.key())
	})
}

func (recorded packageState) key() string {
	return string(recorded.Kind) + "\x00" + recorded.Name + "\x00" + recorded.Version
}

func message(texts *localization.Strings, packages []brew.Package) (string, string) {
	if len(packages) == 1 {
		currentPackage := packages[0]
		return texts.NotificationUpdateTitle(), texts.NotificationPackage(
			currentPackage.Name,
			currentPackage.CurrentVersion,
		)
	}

	title := texts.NotificationUpdatesTitle(len(packages))
	visibleNames := make([]string, 0, 3)
	for index, currentPackage := range packages {
		if index == 3 {
			break
		}
		visibleNames = append(visibleNames, currentPackage.Name)
	}
	remaining := len(packages) - len(visibleNames)
	return title, texts.NotificationPackages(strings.Join(visibleNames, ", "), remaining)
}
