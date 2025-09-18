package versiondiff

import (
	"github.com/hashicorp/go-version"
)

const (
	// DirectionSame is the direction of the version diff when the from and to versions are the same
	DirectionSame = "same"
	// DirectionUpgrade is the direction of the version diff when the from version is less than the to version
	DirectionUpgrade = "upgrade"
	// DirectionDowngrade is the direction of the version diff when the from version is greater than the to version
	DirectionDowngrade = "downgrade"
	// DirectionUnknown is the direction of the version diff when the from and to versions are unknown
	DirectionUnknown = "unknown"
)

// VersionDiff represents the difference between two versions
type VersionDiff struct {
	From *version.Version
	To   *version.Version
}

// IsSameVersion checks if the from and to versions are the same
func (v *VersionDiff) IsSameVersion() bool {
	return v.From.Core().Equal(v.To.Core())
}

// IsUpgrade checks if the from version is less than the to version
func (v *VersionDiff) IsUpgrade() bool {
	return v.To.Core().GreaterThan(v.From.Core())
}

// IsDowngrade checks if the from version is greater than the to version
func (v *VersionDiff) IsDowngrade() bool {
	return v.To.Core().LessThan(v.From.Core())
}

// HasMajorChange checks if the from version is different from the to version
func (v *VersionDiff) HasMajorChange() bool {
	return v.To.Core().Segments()[0] != v.From.Core().Segments()[0]
}

// HasMinorChange checks if the from version is different from the to version
func (v *VersionDiff) HasMinorChange() bool {
	return v.To.Core().Segments()[1] != v.From.Core().Segments()[1]
}

// HasPatchChange checks if the from version is different from the to version
func (v *VersionDiff) HasPatchChange() bool {
	return v.To.Core().Segments()[2] != v.From.Core().Segments()[2]
}

// Direction gets the direction of the version diff as a string
func (v *VersionDiff) Direction() string {
	if v.IsSameVersion() {
		return DirectionSame
	}
	if v.IsUpgrade() {
		return DirectionUpgrade
	}
	if v.IsDowngrade() {
		return DirectionDowngrade
	}
	return DirectionUnknown
}

// DirectionEmoji gets the direction of the version diff as an emoji
func (v *VersionDiff) DirectionEmoji() string {
	switch v.Direction() {
	case DirectionSame:
		return "🔄"
	case DirectionUpgrade:
		return "⬆️"
	case DirectionDowngrade:
		return "⬇️"
	default:
		return "❓"
	}
}
