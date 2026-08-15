package notification

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
// notifies about package versions that were not present in that set.
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

func (service *Service) Handle(result brew.Result) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	previous, err := loadState(service.statePath)
	if err != nil {
		return err
	}

	current := packageStates(result.Packages)
	if equalPackageStates(previous.Packages, current) {
		return nil
	}

	newPackages := newlyAvailable(result.Packages, previous.Packages)
	if err := saveState(service.statePath, state{
		Version:  stateVersion,
		Packages: current,
	}); err != nil {
		return err
	}

	if len(newPackages) > 0 && service.sender != nil {
		title, body := message(service.texts, newPackages)
		service.sender.Send(title, body)
	}
	return nil
}

func loadState(statePath string) (state, error) {
	file, err := os.Open(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return state{Version: stateVersion}, nil
	}
	if err != nil {
		return state{}, fmt.Errorf("notification state could not be opened: %w", err)
	}
	defer file.Close()

	var saved state
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&saved); err != nil {
		return state{}, fmt.Errorf("notification state could not be read: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return state{}, err
	}
	if saved.Version != stateVersion {
		return state{}, fmt.Errorf("unknown notification state version: %d", saved.Version)
	}
	sortPackageStates(saved.Packages)
	return saved, nil
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
	sort.Slice(states, func(left, right int) bool {
		return states[left].key() < states[right].key()
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
