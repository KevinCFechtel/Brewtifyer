package brew

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

type outdatedDocument struct {
	Formulae []outdatedEntry `json:"formulae"`
	Casks    []outdatedEntry `json:"casks"`
}

type outdatedEntry struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	Pinned            bool     `json:"pinned"`
}

// ParseOutdated parses the stable JSON v2 output of `brew outdated`.
func ParseOutdated(reader io.Reader) ([]Package, error) {
	decoder := json.NewDecoder(reader)

	var document outdatedDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("%w: output could not be read: %w", ErrInvalidOutput, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("%w: output contains additional JSON data", ErrInvalidOutput)
		}
		return nil, fmt.Errorf("%w: output contains invalid trailing data: %w", ErrInvalidOutput, err)
	}

	packages := make([]Package, 0, len(document.Formulae)+len(document.Casks))
	appendEntries := func(entries []outdatedEntry, kind Kind) error {
		for _, entry := range entries {
			if strings.TrimSpace(entry.Name) == "" {
				return fmt.Errorf("%w: reported a %s without a name", ErrInvalidOutput, kind)
			}
			if strings.TrimSpace(entry.CurrentVersion) == "" {
				return fmt.Errorf("%w: reported no current version for %q", ErrInvalidOutput, entry.Name)
			}

			packages = append(packages, Package{
				Name:              entry.Name,
				Kind:              kind,
				InstalledVersions: slices.Clone(entry.InstalledVersions),
				CurrentVersion:    entry.CurrentVersion,
				Pinned:            entry.Pinned,
			})
		}
		return nil
	}

	if err := appendEntries(document.Formulae, Formula); err != nil {
		return nil, err
	}
	if err := appendEntries(document.Casks, Cask); err != nil {
		return nil, err
	}

	slices.SortFunc(packages, func(left, right Package) int {
		if left.Kind != right.Kind {
			return strings.Compare(string(left.Kind), string(right.Kind))
		}
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})

	return packages, nil
}
