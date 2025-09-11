package config

import "fmt"

// Cluster represents the Solana cluster configuration
type Cluster struct {
	// Name is the Solana cluster this validator is running on. One of mainnet-beta or testnet
	Name string `koanf:"name"`
}

// SetDefaults sets default values for the cluster configuration
func (c *Cluster) SetDefaults() {
	if c.Name == "" {
		c.Name = "testnet"
	}
}

// Validate validates the cluster configuration
func (c *Cluster) Validate() error {
	validClusters := []string{"mainnet-beta", "testnet"}
	validCluster := false
	for _, cluster := range validClusters {
		if c.Name == cluster {
			validCluster = true
			break
		}
	}
	if !validCluster {
		return fmt.Errorf("cluster.name must be one of %v, got: %s", validClusters, c.Name)
	}

	return nil
}
