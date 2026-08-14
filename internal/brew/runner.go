package brew

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandOutput preserves stdout and stderr separately because Homebrew emits
// warnings on stderr while still returning valid JSON on stdout.
type CommandOutput struct {
	Stdout string
	Stderr string
}

// Runner executes one Homebrew command without involving a shell.
type Runner interface {
	Run(ctx context.Context, executable string, args ...string) (CommandOutput, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, executable string, args ...string) (CommandOutput, error) {
	command := exec.CommandContext(ctx, executable, args...)
	command.Env = append(os.Environ(),
		"HOMEBREW_NO_COLOR=1",
		"HOMEBREW_NO_EMOJI=1",
		"HOMEBREW_NO_ENV_HINTS=1",
		"NO_COLOR=1",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	output := CommandOutput{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return output, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return output, ctxErr
	}

	details := strings.TrimSpace(output.Stderr)
	if len(details) > 600 {
		details = details[:600] + "…"
	}
	if details == "" {
		return output, err
	}
	return output, fmt.Errorf("%w: %s", err, details)
}
