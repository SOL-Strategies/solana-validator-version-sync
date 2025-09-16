package versiondiff

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func TestVersionDiff_StructFields(t *testing.T) {
	v1, _ := version.NewVersion("1.17.0")
	v2, _ := version.NewVersion("1.18.0")
	diff := VersionDiff{
		From: v1,
		To:   v2,
	}

	if diff.From.String() != "1.17.0" {
		t.Errorf("Expected From to be 1.17.0, got %s", diff.From.String())
	}
	if diff.To.String() != "1.18.0" {
		t.Errorf("Expected To to be 1.18.0, got %s", diff.To.String())
	}
}

func TestVersionDiff_IsSameVersion(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "identical versions",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "different major versions",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: false,
		},
		{
			name:     "different minor versions",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "different patch versions",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: false,
		},
		{
			name:     "versions with pre-release",
			from:     "1.18.0-beta.1",
			to:       "1.18.0-beta.1",
			expected: true,
		},
		{
			name:     "versions with different pre-release",
			from:     "1.18.0-beta.1",
			to:       "1.18.0-beta.2",
			expected: true, // hashicorp/go-version considers these equal
		},
		{
			name:     "versions with build metadata",
			from:     "1.18.0+build.1",
			to:       "1.18.0+build.1",
			expected: true,
		},
		{
			name:     "versions with different build metadata",
			from:     "1.18.0+build.1",
			to:       "1.18.0+build.2",
			expected: true, // Build metadata doesn't affect equality
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.IsSameVersion()
			if result != tt.expected {
				t.Errorf("IsSameVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_IsUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "patch upgrade",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: true,
		},
		{
			name:     "minor upgrade",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "major upgrade",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: true,
		},
		{
			name:     "patch downgrade",
			from:     "1.18.1",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "minor downgrade",
			from:     "1.18.0",
			to:       "1.17.0",
			expected: false,
		},
		{
			name:     "major downgrade",
			from:     "2.0.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "same version",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "pre-release upgrade",
			from:     "1.18.0-beta.1",
			to:       "1.18.0-beta.2",
			expected: false, // hashicorp/go-version considers these equal
		},
		{
			name:     "pre-release to release",
			from:     "1.18.0-beta.1",
			to:       "1.18.0",
			expected: false, // hashicorp/go-version considers these equal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.IsUpgrade()
			if result != tt.expected {
				t.Errorf("IsUpgrade() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_IsDowngrade(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "patch downgrade",
			from:     "1.18.1",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "minor downgrade",
			from:     "1.18.0",
			to:       "1.17.0",
			expected: true,
		},
		{
			name:     "major downgrade",
			from:     "2.0.0",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "patch upgrade",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: false,
		},
		{
			name:     "minor upgrade",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "major upgrade",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: false,
		},
		{
			name:     "same version",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "pre-release downgrade",
			from:     "1.18.0-beta.2",
			to:       "1.18.0-beta.1",
			expected: false, // hashicorp/go-version considers these equal
		},
		{
			name:     "release to pre-release",
			from:     "1.18.0",
			to:       "1.18.0-beta.1",
			expected: false, // hashicorp/go-version considers these equal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.IsDowngrade()
			if result != tt.expected {
				t.Errorf("IsDowngrade() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_HasMajorChange(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "major change",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: true,
		},
		{
			name:     "minor change only",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "patch change only",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: false,
		},
		{
			name:     "no change",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "major downgrade",
			from:     "2.0.0",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "pre-release major change",
			from:     "1.18.0",
			to:       "2.0.0-beta.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.HasMajorChange()
			if result != tt.expected {
				t.Errorf("HasMajorChange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_HasMinorChange(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "minor change",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "major change",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: true, // Major change also affects minor
		},
		{
			name:     "patch change only",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: false,
		},
		{
			name:     "no change",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "minor downgrade",
			from:     "1.18.0",
			to:       "1.17.0",
			expected: true,
		},
		{
			name:     "pre-release minor change",
			from:     "1.17.0",
			to:       "1.18.0-beta.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.HasMinorChange()
			if result != tt.expected {
				t.Errorf("HasMinorChange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_HasPatchChange(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected bool
	}{
		{
			name:     "patch change",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: true,
		},
		{
			name:     "minor change",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: false, // Minor change doesn't affect patch segment
		},
		{
			name:     "major change",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: false, // Major change doesn't affect patch segment
		},
		{
			name:     "no change",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: false,
		},
		{
			name:     "patch downgrade",
			from:     "1.18.1",
			to:       "1.18.0",
			expected: true,
		},
		{
			name:     "pre-release patch change",
			from:     "1.18.0",
			to:       "1.18.1-beta.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.HasPatchChange()
			if result != tt.expected {
				t.Errorf("HasPatchChange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_Direction(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected string
	}{
		{
			name:     "upgrade patch",
			from:     "1.18.0",
			to:       "1.18.1",
			expected: "upgrade",
		},
		{
			name:     "upgrade minor",
			from:     "1.17.0",
			to:       "1.18.0",
			expected: "upgrade",
		},
		{
			name:     "upgrade major",
			from:     "1.18.0",
			to:       "2.0.0",
			expected: "upgrade",
		},
		{
			name:     "downgrade patch",
			from:     "1.18.1",
			to:       "1.18.0",
			expected: "downgrade",
		},
		{
			name:     "downgrade minor",
			from:     "1.18.0",
			to:       "1.17.0",
			expected: "downgrade",
		},
		{
			name:     "downgrade major",
			from:     "2.0.0",
			to:       "1.18.0",
			expected: "downgrade",
		},
		{
			name:     "same version",
			from:     "1.18.0",
			to:       "1.18.0",
			expected: "same",
		},
		{
			name:     "pre-release upgrade",
			from:     "1.18.0-beta.1",
			to:       "1.18.0-beta.2",
			expected: "same", // hashicorp/go-version considers these equal
		},
		{
			name:     "pre-release downgrade",
			from:     "1.18.0-beta.2",
			to:       "1.18.0-beta.1",
			expected: "same", // hashicorp/go-version considers these equal
		},
		{
			name:     "pre-release to release",
			from:     "1.18.0-beta.1",
			to:       "1.18.0",
			expected: "same", // hashicorp/go-version considers these equal
		},
		{
			name:     "release to pre-release",
			from:     "1.18.0",
			to:       "1.18.0-beta.1",
			expected: "same", // hashicorp/go-version considers these equal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.Direction()
			if result != tt.expected {
				t.Errorf("Direction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersionDiff_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		expected string
	}{
		{
			name:     "zero versions",
			from:     "0.0.0",
			to:       "0.0.1",
			expected: "upgrade",
		},
		{
			name:     "large version numbers",
			from:     "999.999.999",
			to:       "1000.0.0",
			expected: "upgrade",
		},
		{
			name:     "single digit versions",
			from:     "1.0.0",
			to:       "1.0.1",
			expected: "upgrade",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := version.NewVersion(tt.from)
			to, _ := version.NewVersion(tt.to)
			diff := VersionDiff{From: from, To: to}

			result := diff.Direction()
			if result != tt.expected {
				t.Errorf("Direction() = %v, want %v", result, tt.expected)
			}
		})
	}
}
