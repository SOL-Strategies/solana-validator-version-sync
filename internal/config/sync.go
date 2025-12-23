package config

import (
	"github.com/sol-strategies/solana-validator-version-sync/internal/sync_commands"
)

// Sync represents the version sync configuration
type Sync struct {
	// EnabledWhenActive enables sync when the validator is active
	EnabledWhenActive bool `koanf:"enabled_when_active"`
	// EnabledWhenNoActiveLeaderInGossip enables sync when there is no active leader in gossip
	EnabledWhenNoActiveLeaderInGossip bool `koanf:"enabled_when_no_active_leader_in_gossip"`
	// EnableSFDPCompliance enables SFDP compliance checking
	EnableSFDPCompliance bool `koanf:"enable_sfdp_compliance"`
	// Commands are the commands to run when there is a version change
	Commands []sync_commands.Command `koanf:"commands"`
}

// SetDefaults sets default values for the sync configuration
func (s *Sync) SetDefaults() {
	// This method is kept for any other sync-specific defaults that might be needed
}

// Validate validates the sync configuration
func (s *Sync) Validate() error {
	//This function is kept for any other sync-specific validation that might be needed
	return nil
}
