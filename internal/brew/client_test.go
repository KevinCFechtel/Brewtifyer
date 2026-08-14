package brew

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type runnerCall struct {
	executable string
	args       []string
}

type fakeRunner struct {
	outputs []CommandOutput
	errors  []error
	calls   []runnerCall
}

func (runner *fakeRunner) Run(_ context.Context, executable string, args ...string) (CommandOutput, error) {
	runner.calls = append(runner.calls, runnerCall{executable: executable, args: args})
	index := len(runner.calls) - 1
	return runner.outputs[index], runner.errors[index]
}

func TestClientCheckUsesCachedDataWhenUpdateFails(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{
		outputs: []CommandOutput{
			{},
			{Stdout: `{"formulae":[],"casks":[]}`},
		},
		errors: []error{errors.New("offline"), nil},
	}
	checkedAt := time.Date(2026, time.August, 14, 12, 30, 0, 0, time.Local)
	client := NewClient("/opt/homebrew/bin/brew")
	client.runner = runner
	client.now = func() time.Time { return checkedAt }

	result, err := client.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if result.CheckedAt != checkedAt {
		t.Errorf("CheckedAt = %v, want %v", result.CheckedAt, checkedAt)
	}
	if result.Warning == "" {
		t.Error("Warning is empty, want update warning")
	}

	wantCalls := []runnerCall{
		{executable: "/opt/homebrew/bin/brew", args: []string{"update-if-needed"}},
		{executable: "/opt/homebrew/bin/brew", args: []string{"outdated", "--json=v2"}},
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Errorf("calls = %#v, want %#v", runner.calls, wantCalls)
	}
}

func TestClientCheckReturnsOutdatedError(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{
		outputs: []CommandOutput{{}, {}},
		errors:  []error{nil, errors.New("brew failed")},
	}
	client := NewClient("/brew")
	client.runner = runner

	_, err := client.Check(context.Background())
	if err == nil {
		t.Fatal("Check() error = nil, want error")
	}
}
