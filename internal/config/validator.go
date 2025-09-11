package config

import (
	"fmt"
	"os"
)

// Validator represents the validator configuration
type Validator struct {
	// Client is the solana validator client - one of: agave, jito-solana, firedancer
	Client string `koanf:"client"`
	// RPCURL is the URL of the validator's RPC endpoint
	RPCURL string `koanf:"rpc_url"`
	// Identities are the paths to the active and passive identity keyfiles
	Identities Identities `koanf:"identities"`
}

// Identities represents the validator identity configuration
type Identities struct {
	// Active is the path to the active identity keyfile
	Active string `koanf:"active"`
	// Passive is the path to the passive identity keyfile
	Passive string `koanf:"passive"`
	// ActiveKeyPair is the loaded active keypair (simplified for now)
	ActiveKeyPair interface{} `koanf:"-"`
	// PassiveKeyPair is the loaded passive keypair (simplified for now)
	PassiveKeyPair interface{} `koanf:"-"`
	// ActiveKeyPairFile is the file path for active keypair
	ActiveKeyPairFile string `koanf:"-"`
	// PassiveKeyPairFile is the file path for passive keypair
	PassiveKeyPairFile string `koanf:"-"`
}

// Load loads the identity keypairs from files
func (i *Identities) Load() error {
	// Load active identity
	if i.Active != "" {
		i.ActiveKeyPairFile = i.Active
		// For now, just store the file path - keypair loading can be implemented later
		i.ActiveKeyPair = i.Active
	}

	// Load passive identity
	if i.Passive != "" {
		i.PassiveKeyPairFile = i.Passive
		// For now, just store the file path - keypair loading can be implemented later
		i.PassiveKeyPair = i.Passive
	}

	return nil
}

// SetDefaults sets default values for the validator configuration
func (v *Validator) SetDefaults() {
	if v.RPCURL == "" {
		v.RPCURL = "http://127.0.0.1:8899"
	}
}

// Validate validates the validator configuration
func (v *Validator) Validate() error {
	// Validate client
	validClients := []string{"agave", "jito-solana", "firedancer"}
	validClient := false
	for _, client := range validClients {
		if v.Client == client {
			validClient = true
			break
		}
	}
	if !validClient {
		return fmt.Errorf("validator.client must be one of %v, got: %s", validClients, v.Client)
	}

	// Validate RPC URL
	if v.RPCURL == "" {
		return fmt.Errorf("validator.rpc_url is required")
	}

	// Validate identities
	if v.Identities.Active == "" {
		return fmt.Errorf("validator.identities.active is required")
	}
	if v.Identities.Passive == "" {
		return fmt.Errorf("validator.identities.passive is required")
	}

	// Check if identity files exist
	if _, err := os.Stat(v.Identities.Active); os.IsNotExist(err) {
		return fmt.Errorf("active identity file does not exist: %s", v.Identities.Active)
	}
	if _, err := os.Stat(v.Identities.Passive); os.IsNotExist(err) {
		return fmt.Errorf("passive identity file does not exist: %s", v.Identities.Passive)
	}

	return nil
}
