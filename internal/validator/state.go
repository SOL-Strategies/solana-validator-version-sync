package validator

import "github.com/hashicorp/go-version"

// State represents the state of the validator
type State struct {
	Cluster           string
	VersionString     string
	HealthStatus      string
	IdentityPublicKey string
	Version           *version.Version
}
