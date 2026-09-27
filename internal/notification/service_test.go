package notification

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KevinCFechtel/Brewtifyer/internal/brew"
	"github.com/KevinCFechtel/Brewtifyer/internal/localization"
)

type sentMessage struct {
	title string
	body  string
}

type recordingSender struct {
	messages []sentMessage
}

func (sender *recordingSender) Send(title, body string) {
	sender.messages = append(sender.messages, sentMessage{title: title, body: body})
}

func TestServiceDeduplicatesAcrossRestarts(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "Brewtifyer", "notification-state.json")
	sender := &recordingSender{}
	service := NewService(statePath, sender, localization.MustNew("de"))
	updates := resultWith(packageUpdate("go", brew.Formula, "1.26.6"))

	if _, err := service.Handle(updates); err != nil {
		t.Fatalf("handle first result: %v", err)
	}
	if _, err := service.Handle(updates); err != nil {
		t.Fatalf("handle duplicate result: %v", err)
	}

	restartedService := NewService(statePath, sender, localization.MustNew("de"))
	if _, err := restartedService.Handle(updates); err != nil {
		t.Fatalf("handle result after restart: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent %d notifications, want 1", len(sender.messages))
	}
}

func TestServiceOnlyNotifiesAboutNewUpdates(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "notification-state.json")
	sender := &recordingSender{}
	service := NewService(statePath, sender, localization.MustNew("de"))
	goUpdate := packageUpdate("go", brew.Formula, "1.26.6")
	nodeUpdate := packageUpdate("node", brew.Formula, "25.0.0")

	if _, err := service.Handle(resultWith(goUpdate, nodeUpdate)); err != nil {
		t.Fatalf("handle initial result: %v", err)
	}
	if _, err := service.Handle(resultWith(goUpdate)); err != nil {
		t.Fatalf("handle removed update: %v", err)
	}

	rustUpdate := packageUpdate("rust", brew.Formula, "2.0.0")
	if _, err := service.Handle(resultWith(goUpdate, rustUpdate)); err != nil {
		t.Fatalf("handle new update: %v", err)
	}

	if len(sender.messages) != 2 {
		t.Fatalf("sent %d notifications, want 2", len(sender.messages))
	}
	latest := sender.messages[1]
	if latest.title != "Homebrew-Update verfügbar" {
		t.Fatalf("title = %q", latest.title)
	}
	if !strings.Contains(latest.body, "rust") || strings.Contains(latest.body, "go") {
		t.Fatalf("body = %q, want only the newly available package", latest.body)
	}
}

func TestServiceTreatsNewTargetVersionAsNewUpdate(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "notification-state.json")
	sender := &recordingSender{}
	service := NewService(statePath, sender, localization.MustNew("de"))

	if _, err := service.Handle(resultWith(packageUpdate("go", brew.Formula, "1.26.6"))); err != nil {
		t.Fatalf("handle first version: %v", err)
	}
	if _, err := service.Handle(resultWith(packageUpdate("go", brew.Formula, "1.26.7"))); err != nil {
		t.Fatalf("handle next version: %v", err)
	}

	if len(sender.messages) != 2 {
		t.Fatalf("sent %d notifications, want 2", len(sender.messages))
	}
	if !strings.Contains(sender.messages[1].body, "1.26.7") {
		t.Fatalf("body = %q, want new target version", sender.messages[1].body)
	}
}

func TestServiceNotifiesWhenUpdateReappears(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "notification-state.json")
	sender := &recordingSender{}
	service := NewService(statePath, sender, localization.MustNew("de"))
	updates := resultWith(packageUpdate("go", brew.Formula, "1.26.6"))

	if _, err := service.Handle(updates); err != nil {
		t.Fatalf("handle first result: %v", err)
	}
	if _, err := service.Handle(resultWith()); err != nil {
		t.Fatalf("handle empty result: %v", err)
	}
	if _, err := service.Handle(updates); err != nil {
		t.Fatalf("handle reappeared update: %v", err)
	}

	if len(sender.messages) != 2 {
		t.Fatalf("sent %d notifications, want 2", len(sender.messages))
	}
}

func TestServicePersistsStatePrivately(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "Brewtifyer", "notification-state.json")
	service := NewService(statePath, &recordingSender{}, localization.MustNew("de"))
	if _, err := service.Handle(resultWith(packageUpdate("go", brew.Formula, "1.26.6"))); err != nil {
		t.Fatalf("handle result: %v", err)
	}

	information, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if permissions := information.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("state permissions = %o, want 600", permissions)
	}
}

// The state file is a deduplication cache. An unreadable one must be rebuilt
// rather than reported as a failure: returning an error here would skip
// saveState, so notifications would stay broken for good until someone deleted
// the file by hand. Paying one duplicate notification is the cheaper trade.
func TestServiceRecoversFromInvalidState(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "notification-state.json")
	if err := os.WriteFile(statePath, []byte("not json\n"), 0o600); err != nil {
		t.Fatalf("write invalid state: %v", err)
	}
	sender := &recordingSender{}
	service := NewService(statePath, sender, localization.MustNew("de"))

	note, err := service.Handle(resultWith(packageUpdate("go", brew.Formula, "1.26.6")))
	if err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
	if note == "" {
		t.Error("note is empty, want an explanation that the state was discarded")
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent %d notifications, want 1", len(sender.messages))
	}

	// The rebuilt state must be usable, so a second identical result is quiet.
	if _, err := service.Handle(resultWith(packageUpdate("go", brew.Formula, "1.26.6"))); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent %d notifications after recovery, want 1", len(sender.messages))
	}
}

// A state file written by a newer Brewtifyer must not wedge an older one.
func TestServiceRecoversFromUnknownStateVersion(t *testing.T) {
	t.Parallel()

	statePath := filepath.Join(t.TempDir(), "notification-state.json")
	future := `{"version":99,"packages":[{"name":"go","kind":"formula","version":"1.26.6"}]}`
	if err := os.WriteFile(statePath, []byte(future), 0o600); err != nil {
		t.Fatalf("write future state: %v", err)
	}
	sender := &recordingSender{}
	service := NewService(statePath, sender, localization.MustNew("de"))

	note, err := service.Handle(resultWith(packageUpdate("go", brew.Formula, "1.26.6")))
	if err != nil {
		t.Fatalf("Handle() error = %v, want nil", err)
	}
	if !strings.Contains(note, "99") {
		t.Errorf("note = %q, want it to name the unsupported version", note)
	}

	// The file must have been rewritten at the supported version.
	content, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(content), `"version": 1`) {
		t.Errorf("state = %s, want it rewritten at version 1", content)
	}
}

func TestPluralMessageListsAtMostThreePackages(t *testing.T) {
	t.Parallel()

	packages := []brew.Package{
		packageUpdate("go", brew.Formula, "1"),
		packageUpdate("node", brew.Formula, "2"),
		packageUpdate("rust", brew.Formula, "3"),
		packageUpdate("firefox", brew.Cask, "4"),
	}
	title, body := message(localization.MustNew("de"), packages)
	if title != "4 neue Homebrew-Updates" {
		t.Fatalf("title = %q", title)
	}
	if body != "Neu verfügbar für: go, node, rust und 1 weiteres Update." {
		t.Fatalf("body = %q", body)
	}
}

func TestMessageUsesSelectedLanguage(t *testing.T) {
	t.Parallel()

	title, body := message(localization.MustNew("en"), []brew.Package{
		packageUpdate("go", brew.Formula, "1.26.6"),
		packageUpdate("node", brew.Formula, "25.0.0"),
	})
	if title != "2 new Homebrew updates" {
		t.Fatalf("title = %q", title)
	}
	if body != "Newly available for: go, node." {
		t.Fatalf("body = %q", body)
	}
}

func resultWith(packages ...brew.Package) brew.Result {
	return brew.Result{Packages: packages}
}

func packageUpdate(name string, kind brew.Kind, version string) brew.Package {
	return brew.Package{
		Name:           name,
		Kind:           kind,
		CurrentVersion: version,
	}
}
