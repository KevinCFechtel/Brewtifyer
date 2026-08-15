package brew

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultUpdateTimeout   = 5 * time.Minute
	defaultOutdatedTimeout = 2 * time.Minute
)

// Client asks a locally installed Homebrew for available updates.
type Client struct {
	path            string
	runner          Runner
	now             func() time.Time
	updateTimeout   time.Duration
	outdatedTimeout time.Duration
}

func NewClient(path string) *Client {
	return &Client{
		path:            path,
		runner:          ExecRunner{},
		now:             time.Now,
		updateTimeout:   defaultUpdateTimeout,
		outdatedTimeout: defaultOutdatedTimeout,
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
		return Result{}, fmt.Errorf("Homebrew updates could not be queried: %w", err)
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
		result.Warning = fmt.Sprintf("Homebrew metadata could not be updated: %v", updateErr)
	}
	return result, nil
}
