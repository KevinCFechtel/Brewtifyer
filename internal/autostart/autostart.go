// Package autostart manages Brewtifyer's macOS login item through
// SMAppService. Non-darwin builds report the feature as unsupported.
package autostart

import "fmt"

// Status describes whether macOS can and may launch Brewtifyer at login.
type Status uint8

const (
	Unsupported Status = iota
	Disabled
	Enabled
	RequiresApproval
	NotFound
)

type Controller interface {
	Status() (Status, error)
	SetEnabled(enabled bool) (Status, error)
	OpenSettings() error
}

// Native status codes shared with the Objective-C bridge in native_darwin.m.
// The two lists must stay in sync; statusFromCode is the single place that
// translates between them, which keeps it testable without cgo.
const (
	nativeError            = -1
	nativeUnsupported      = 0
	nativeDisabled         = 1
	nativeEnabled          = 2
	nativeRequiresApproval = 3
	nativeNotFound         = 4
)

// statusFromCode converts a native status code into a Status.
func statusFromCode(code int) (Status, error) {
	switch code {
	case nativeUnsupported:
		return Unsupported, nil
	case nativeDisabled:
		return Disabled, nil
	case nativeEnabled:
		return Enabled, nil
	case nativeRequiresApproval:
		return RequiresApproval, nil
	case nativeNotFound:
		return NotFound, nil
	default:
		return NotFound, fmt.Errorf("unknown native launch-at-login status: %d", code)
	}
}
