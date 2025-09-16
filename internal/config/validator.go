package config

import (
	"fmt"
	"net/url"

	"github.com/gagliardetto/solana-go"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
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
	ActiveKeyPairFile string `koanf:"active"`
	// Passive is the path to the passive identity keyfile
	PassiveKeyPairFile string `koanf:"passive"`
	// ActiveKeyPair is the loaded active keypair
	ActiveKeyPair solana.PrivateKey `koanf:"-"`
	// PassiveKeyPair is the loaded passive keypair
	PassiveKeyPair solana.PrivateKey `koanf:"-"`
}

// Load loads the identity keypairs from files
func (i *Identities) Load() (err error) {

	// Load active identity
	i.ActiveKeyPair, err = solana.PrivateKeyFromSolanaKeygenFile(i.ActiveKeyPairFile)
	if err != nil {
		return fmt.Errorf("failed to load active keypair from %s: %w", i.ActiveKeyPairFile, err)
	}

	// Load passive identity
	i.PassiveKeyPair, err = solana.PrivateKeyFromSolanaKeygenFile(i.PassiveKeyPairFile)
	if err != nil {
		return fmt.Errorf("failed to load passive keypair from %s: %w", i.PassiveKeyPairFile, err)
	}

	return nil
}

// Validate validates the validator configuration
func (v *Validator) Validate() error {
	// Validate client
	err := constants.ValidateClientName(v.Client)
	if err != nil {
		return err
	}

	// Validate RPC URL
	_, err = url.Parse(v.RPCURL)
	if err != nil {
		return fmt.Errorf("validator.rpc_url %s is not a valid URL: %w", v.RPCURL, err)
	}

	return nil
}
