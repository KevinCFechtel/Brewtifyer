package brew

import "errors"

// Sentinel errors let the user interface pick a localized message without
// matching on English error text. Every error returned from this package wraps
// exactly one of them, while the wrapped detail stays intact for the log.
//
// The messages themselves are only ever shown in logs, so they deliberately
// read as technical diagnostics rather than as user-facing copy.
var (
	// ErrNotFound means no usable brew executable could be located.
	ErrNotFound = errors.New("Homebrew executable was not found")
	// ErrQueryFailed means brew ran but did not complete successfully.
	ErrQueryFailed = errors.New("Homebrew could not be queried")
	// ErrInvalidOutput means brew succeeded but its output could not be
	// interpreted. This is the signal that the expected JSON schema changed.
	ErrInvalidOutput = errors.New("Homebrew returned unexpected data")
	// ErrTimeout means brew exceeded the time budget for a command.
	ErrTimeout = errors.New("Homebrew did not respond in time")
)
