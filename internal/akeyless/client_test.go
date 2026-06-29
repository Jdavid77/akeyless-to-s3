package akeyless

import "testing"

func TestIsSecretType(t *testing.T) {
	tests := []struct {
		itemType string
		want     bool
	}{
		{"static-secret", true},
		{"static_secret", true},
		{"dynamic-secret", true},
		{"dynamic_secret", true},
		{"rotated-secret", true},
		{"rotated_secret", true},
		// case-insensitive
		{"STATIC-SECRET", true},
		{"Static-Secret", true},
		// folder and unknowns must not match
		{"folder", false},
		{"vault_secret", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.itemType, func(t *testing.T) {
			if got := IsSecretType(tt.itemType); got != tt.want {
				t.Errorf("IsSecretType(%q) = %v, want %v", tt.itemType, got, tt.want)
			}
		})
	}
}

func TestExtractNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/app/db/password", "password"},
		{"/password", "password"},
		{"password", "password"},
		// trailing slash trimmed before split
		{"/app/db/password/", "password"},
		// single slash — returns empty string (last component of ["", ""])
		{"/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := ExtractNameFromPath(tt.path); got != tt.want {
				t.Errorf("ExtractNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
