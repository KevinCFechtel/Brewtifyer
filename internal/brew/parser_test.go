package brew

import (
	"strings"
	"testing"
)

func TestParseOutdated(t *testing.T) {
	t.Parallel()

	input := `{
  "formulae": [{
    "name": "go",
    "installed_versions": ["1.26.5"],
    "current_version": "1.26.6",
    "pinned": false,
    "pinned_version": null
  }],
  "casks": [{
    "name": "firefox",
    "installed_versions": ["142.0"],
    "current_version": "143.0",
    "pinned": true,
    "pinned_version": "142.0"
  }]
}`

	packages, err := ParseOutdated(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseOutdated() error = %v", err)
	}
	if len(packages) != 2 {
		t.Fatalf("ParseOutdated() returned %d packages, want 2", len(packages))
	}

	if packages[0].Name != "firefox" || packages[0].Kind != Cask || !packages[0].Pinned {
		t.Errorf("first package = %#v, want pinned firefox cask", packages[0])
	}
	if packages[1].Name != "go" || packages[1].Kind != Formula {
		t.Errorf("second package = %#v, want go formula", packages[1])
	}
}

func TestParseOutdatedEmpty(t *testing.T) {
	t.Parallel()

	packages, err := ParseOutdated(strings.NewReader(`{"formulae":[],"casks":[]}`))
	if err != nil {
		t.Fatalf("ParseOutdated() error = %v", err)
	}
	if len(packages) != 0 {
		t.Fatalf("ParseOutdated() returned %d packages, want 0", len(packages))
	}
}

func TestParseOutdatedRejectsTrailingJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseOutdated(strings.NewReader(`{"formulae":[],"casks":[]} {}`))
	if err == nil {
		t.Fatal("ParseOutdated() error = nil, want trailing data error")
	}
}
