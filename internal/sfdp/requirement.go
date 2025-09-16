package sfdp

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

// RequiredVersions represents an SFDP version constraints
type Requirements struct {
	Epoch                      int    `json:"epoch"`
	Cluster                    string `json:"cluster"`
	AgaveMinVersion            string `json:"agave_min_version"`
	AgaveMaxVersion            string `json:"agave_max_version"`
	FiredancerMinVersion       string `json:"firedancer_min_version"`
	FiredancerMaxVersion       string `json:"firedancer_max_version"`
	InheritedFromPreviousEpoch bool   `json:"inherited_from_previous_epoch"`

	Client            string
	ConstraintsString string
	Constraints       version.Constraints
	MaxVersion        *version.Version
	MinVersion        *version.Version
	HasMaxVersion     bool
	HasMinVersion     bool
}

// SetClient sets the client and limits for it
func (r *Requirements) SetClient(client string) (err error) {
	var minVersion string
	var maxVersion string

	switch client {
	case constants.ClientNameAgave, constants.ClientNameJitoSolana:
		r.Client = constants.ClientNameAgave
		minVersion = r.AgaveMinVersion
		maxVersion = r.AgaveMaxVersion
	case constants.ClientNameFiredancer:
		r.Client = client
		minVersion = r.FiredancerMinVersion
		maxVersion = r.FiredancerMaxVersion
	default:
		return fmt.Errorf("invalid client: %s", client)
	}

	// build a constraints string
	var constraintsStrings = []string{}
	if minVersion != "" {
		r.HasMinVersion = true
		r.MinVersion, err = version.NewVersion(minVersion)
		if err != nil {
			return fmt.Errorf("failed to parse min version: %w", err)
		}
		constraintsStrings = append(constraintsStrings, fmt.Sprintf(">= %s", minVersion))
	}
	if maxVersion != "" {
		r.HasMaxVersion = true
		r.MaxVersion, err = version.NewVersion(maxVersion)
		if err != nil {
			return fmt.Errorf("failed to parse max version: %w", err)
		}
		constraintsStrings = append(constraintsStrings, fmt.Sprintf("<= %s", maxVersion))
	}

	// set it
	r.ConstraintsString = strings.Join(constraintsStrings, ",")

	// build constraints from string
	r.Constraints, err = version.NewConstraint(r.ConstraintsString)
	if err != nil {
		return fmt.Errorf("failed to parse constraints: %w", err)
	}

	return nil
}
