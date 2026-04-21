package config

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sync_commands"
)

var syncValidationLogger = log.WithPrefix("config")

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
	for i, command := range s.Commands {
		if len(command.Environment) == 0 || command.InheritEnvironment {
			continue
		}

		commandName := command.Name
		if commandName == "" {
			commandName = fmt.Sprintf("command[%d]", i)
		}

		syncValidationLogger.Warn(
			"sync command defines environment with inherit_environment=false - only the explicit environment block will be passed to the child process",
			"command", commandName,
			"command_index", i,
			"inherit_environment", command.InheritEnvironment,
		)
	}

	return nil
}
