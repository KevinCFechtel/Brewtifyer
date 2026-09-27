package brew

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// Every error leaving this package must wrap a sentinel, otherwise the user
// interface silently falls back to a generic message.
func TestLocateErrorsWrapErrNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configuredPath string
	}{
		{name: "invalid configured path", configuredPath: filepath.Join(t.TempDir(), "missing")},
		{name: "no path configured", configuredPath: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// With no configured path Locate may legitimately find a real brew
			// on the developer machine, so only assert when it fails.
			_, err := Locate(test.configuredPath)
			if err == nil {
				if test.configuredPath != "" {
					t.Fatal("Locate() error = nil, want error")
				}
				t.Skip("Homebrew is installed on this machine")
			}
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("Locate() error = %v, want it to wrap ErrNotFound", err)
			}
		})
	}
}

func TestParseOutdatedErrorsWrapErrInvalidOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "malformed json", input: `{"formulae":`},
		{name: "trailing json", input: `{"formulae":[],"casks":[]} {}`},
		{name: "formula without name", input: `{"formulae":[{"current_version":"1"}],"casks":[]}`},
		{
			name:  "formula without current version",
			input: `{"formulae":[{"name":"go"}],"casks":[]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseOutdated(strings.NewReader(test.input))
			if err == nil {
				t.Fatal("ParseOutdated() error = nil, want error")
			}
			if !errors.Is(err, ErrInvalidOutput) {
				t.Fatalf("ParseOutdated() error = %v, want it to wrap ErrInvalidOutput", err)
			}
		})
	}
}

func TestCheckClassifiesTimeoutSeparatelyFromFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		outdatedErr error
		want        error
	}{
		{name: "deadline exceeded", outdatedErr: context.DeadlineExceeded, want: ErrTimeout},
		{name: "cancelled", outdatedErr: context.Canceled, want: ErrTimeout},
		{name: "command failed", outdatedErr: errors.New("exit status 1"), want: ErrQueryFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			runner := &fakeRunner{
				outputs: []CommandOutput{{}, {}},
				errors:  []error{nil, test.outdatedErr},
			}
			client := NewClient("/brew")
			client.runner = runner

			_, err := client.Check(context.Background())
			if !errors.Is(err, test.want) {
				t.Fatalf("Check() error = %v, want it to wrap %v", err, test.want)
			}
			// The underlying cause must survive for the log.
			if !errors.Is(err, test.outdatedErr) {
				t.Errorf("Check() error = %v, want it to wrap the cause %v", err, test.outdatedErr)
			}
		})
	}
}

// A successful brew run whose output cannot be parsed must be reported as an
// output problem, not as a query failure: it is the signal that the JSON schema
// changed and that Brewtifyer needs updating.
func TestCheckReportsUnparsableOutputAsInvalid(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{
		outputs: []CommandOutput{{}, {Stdout: "not json at all"}},
		errors:  []error{nil, nil},
	}
	client := NewClient("/brew")
	client.runner = runner

	_, err := client.Check(context.Background())
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("Check() error = %v, want it to wrap ErrInvalidOutput", err)
	}
}
