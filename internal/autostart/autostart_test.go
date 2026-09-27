package autostart

import "testing"

// statusFromCode is the only translation between the Objective-C bridge and Go.
// A drift between the two enums would silently mislabel the login item state,
// so every documented code is pinned here.
func TestStatusFromCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    int
		want    Status
		wantErr bool
	}{
		{name: "unsupported", code: nativeUnsupported, want: Unsupported},
		{name: "disabled", code: nativeDisabled, want: Disabled},
		{name: "enabled", code: nativeEnabled, want: Enabled},
		{name: "requires approval", code: nativeRequiresApproval, want: RequiresApproval},
		{name: "not found", code: nativeNotFound, want: NotFound},
		{name: "error sentinel", code: nativeError, want: NotFound, wantErr: true},
		{name: "unknown code", code: 42, want: NotFound, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			status, err := statusFromCode(test.code)
			if status != test.want {
				t.Errorf("statusFromCode(%d) status = %v, want %v", test.code, status, test.want)
			}
			if (err != nil) != test.wantErr {
				t.Errorf("statusFromCode(%d) error = %v, wantErr = %t", test.code, err, test.wantErr)
			}
		})
	}
}

// An unrecognized code must never be reported as a usable state.
func TestStatusFromCodeNeverReportsEnabledOnError(t *testing.T) {
	t.Parallel()

	for code := -10; code < 20; code++ {
		status, err := statusFromCode(code)
		if err != nil && status == Enabled {
			t.Fatalf("statusFromCode(%d) reported Enabled alongside error %v", code, err)
		}
	}
}
