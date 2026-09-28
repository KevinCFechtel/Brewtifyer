package brew

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultUpdateTimeout   = 5 * time.Minute
	defaultOutdatedTimeout = 2 * time.Minute
	defaultVersionTimeout  = 30 * time.Second
)

// Client asks a locally installed Homebrew for available updates.
type Client struct {
	path            string
	runner          Runner
	now             func() time.Time
	updateTimeout   time.Duration
	outdatedTimeout time.Duration
	versionTimeout  time.Duration
}

func NewClient(path string) *Client {
	return &Client{
		path:            path,
		runner:          ExecRunner{},
		now:             time.Now,
		updateTimeout:   defaultUpdateTimeout,
		outdatedTimeout: defaultOutdatedTimeout,
		versionTimeout:  defaultVersionTimeout,
	}
}

// Check refreshes Homebrew metadata when needed and then queries outdated
// formulae and casks. A failed metadata refresh becomes a warning if cached
// data can still be queried successfully.
func (client *Client) Check(ctx context.Context) (Result, error) {
	updateContext, cancelUpdate := context.WithTimeout(ctx, client.updateTimeout)
	_, updateErr := client.runner.Run(updateContext, client.path, "update-if-needed")
	cancelUpdate()

	outdatedContext, cancelOutdated := context.WithTimeout(ctx, client.outdatedTimeout)
	output, err := client.runner.Run(outdatedContext, client.path, "outdated", "--json=v2")
	cancelOutdated()
	if err != nil {
		return Result{}, classifyRunError(err)
	}

	packages, err := ParseOutdated(strings.NewReader(output.Stdout))
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Packages:  packages,
		CheckedAt: client.now(),
	}
	if updateErr != nil {
		result.Warning = &Warning{
			Kind:  WarningMetadataRefreshFailed,
			Cause: updateErr,
		}
	}
	return result, nil
}

// Version reports the Homebrew version string, for example "4.3.10". It exists
// purely for diagnostics: the version is written to the log at startup so that
// a bug report identifies the Homebrew that produced the output. Brewtifyer
// never gates behaviour on it, because an unexpected schema is detected from
// the output itself through ErrInvalidOutput.
func (client *Client) Version(ctx context.Context) (string, error) {
	versionContext, cancel := context.WithTimeout(ctx, client.versionTimeout)
	defer cancel()

	output, err := client.runner.Run(versionContext, client.path, "--version")
	if err != nil {
		return "", classifyRunError(err)
	}

	// `brew --version` prints several lines; the first carries the version.
	firstLine, _, _ := strings.Cut(output.Stdout, "\n")
	version := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(firstLine), "Homebrew"))
	if version == "" {
		return "", fmt.Errorf("%w: --version printed no version", ErrInvalidOutput)
	}
	return version, nil
}

// classifyRunError maps a failed brew invocation onto a sentinel error so that
// the user interface can localize it. A cancelled parent context is reported as
// a timeout too: from the user's perspective the check simply did not finish.
func classifyRunError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %w", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %w", ErrQueryFailed, err)
}
