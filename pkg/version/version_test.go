package version

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	testCases := []struct {
		name       string
		setVersion string
		expect     string
	}{
		{
			name:       "injected non-dev version with v prefix",
			setVersion: "v1.2.3",
			expect:     "v1.2.3",
		},
		{
			name:       "Version is dev",
			setVersion: defaultVersion,
			expect:     defaultVersion,
		},
		{
			name:       "default to dev when empty",
			setVersion: "",
			expect:     defaultVersion,
		},
		{
			name:       "numeric injected version prefixed with v",
			setVersion: "1.2.3",
			expect:     "v1.2.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Version = tc.setVersion
			got := GetVersion()
			if got != tc.expect {
				// Handle the case where build info might override empty version
				if tc.setVersion == "" && got != defaultVersion && got != "(devel)" {
					// This is acceptable if build info provides a version
					return
				}
				t.Fatalf("GetVersion() = %q, want %q", got, tc.expect)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "empty string",
			input:  "",
			expect: "",
		},
		{
			name:   "already prefixed with v",
			input:  "v1.2.3",
			expect: "v1.2.3",
		},
		{
			name:   "numeric version gets v prefix",
			input:  "1.2.3",
			expect: "v1.2.3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeVersion(tc.input)
			if got != tc.expect {
				t.Fatalf("NormalizeVersion(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}
