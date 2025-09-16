package config

import (
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

// Cluster represents the Solana cluster configuration
type Cluster struct {
	// Name is the Solana cluster this validator is running on. One of mainnet-beta or testnet
	Name string `koanf:"name"`
}

// Validate validates the cluster configuration
func (c *Cluster) Validate() error {
	return constants.ValidateClusterName(c.Name)
}
