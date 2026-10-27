package sfdp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

// RequiredVersions represents an SFDP version constraints
type Requirements struct {
	Epoch                      int     `json:"epoch"`
	Cluster                    string  `json:"cluster"`
	AgaveMinVersion            string  `json:"agave_min_version"`
	AgaveMaxVersion            string  `json:"agave_max_version"`
	FiredancerMinVersion       string  `json:"firedancer_min_version"`
	FiredancerMaxVersion       string  `json:"firedancer_max_version"`
	FiredancerFullMinVersion   *string `json:"firedancer_full_min_version"`
	FiredancerFullMaxVersion   *string `json:"firedancer_full_max_version"`
	firedancerFullMinPresent   bool
	firedancerFullMaxPresent   bool
	InheritedFromPreviousEpoch bool `json:"inherited_from_previous_epoch"`

	Client        string
	Constraints   version.Constraints
	MaxVersion    *version.Version
	MinVersion    *version.Version
	HasMaxVersion bool
	HasMinVersion bool
}

func (r *Requirements) UnmarshalJSON(data []byte) error {
	type Alias Requirements
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	var decoded Alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = Requirements(decoded)
	_, r.firedancerFullMinPresent = fields["firedancer_full_min_version"]
	_, r.firedancerFullMaxPresent = fields["firedancer_full_max_version"]
	return nil
}

// SetClient sets the client and limits for it
func (r *Requirements) SetClient(client string) (err error) {
	var minVersion string
	var maxVersion string
	normalizedClient := constants.NormalizeClientName(client)

	switch normalizedClient {
	case constants.ClientNameAgave, constants.ClientNameJitoSolana, constants.ClientNameRakurai:
		r.Client = constants.ClientNameAgave
		minVersion = r.AgaveMinVersion
		maxVersion = r.AgaveMaxVersion
	case constants.ClientNameFiredancer, constants.ClientNameFireBAM:
		r.Client = client
		if !r.firedancerFullMinPresent || !r.firedancerFullMaxPresent {
			return fmt.Errorf("SFDP response is missing native Firedancer version requirements")
		}
		if r.FiredancerFullMinVersion != nil {
			minVersion = *r.FiredancerFullMinVersion
		}
		if r.FiredancerFullMaxVersion != nil {
			maxVersion = *r.FiredancerFullMaxVersion
		}
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
	constraintsString := strings.Join(constraintsStrings, ",")

	// build constraints from string
	r.Constraints, err = version.NewConstraint(constraintsString)
	if err != nil {
		return fmt.Errorf("failed to parse constraints: %w", err)
	}

	return nil
}
