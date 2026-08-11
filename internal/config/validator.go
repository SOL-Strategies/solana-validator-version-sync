package config

import (
	"fmt"
	"net/url"

	"github.com/gagliardetto/solana-go"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

// Validator represents the validator configuration
type Validator struct {
	// Client is the solana validator client - one of: agave, jito-solana, rakurai-validator, firedancer
	// The legacy alias "rakurai" is also accepted and normalized to "rakurai-validator".
	Client string `koanf:"client"`
	// RPCURL is the URL of the validator's RPC endpoint
	RPCURL string `koanf:"rpc_url"`
	// VersionConstraint is the constraint for the client version
	VersionConstraint string `koanf:"version_constraint"`
	// VoteAccountPublicKey is the vote account used to discover the active validator identity
	VoteAccountPublicKey string `koanf:"vote_account_pubkey"`
	// Identities configures active and optional passive validator identities
	Identities Identities `koanf:"identities"`
}

// Identities represents public-key and legacy keypair validator identity configuration.
type Identities struct {
	// ActiveKeyPairFile is the legacy path to the active identity keyfile
	ActiveKeyPairFile string `koanf:"active"`
	// ActivePublicKey is the active validator identity public key
	ActivePublicKey string `koanf:"active_pubkey"`
	// PassiveKeyPairFile is the legacy path to the passive identity keyfile
	PassiveKeyPairFile string `koanf:"passive"`
	// PassivePublicKey is the passive validator identity public key
	PassivePublicKey string `koanf:"passive_pubkey"`
	// ActiveKeyPair is the loaded active keypair
	ActiveKeyPair solana.PrivateKey `koanf:"-"`
	// PassiveKeyPair is the loaded passive keypair
	PassiveKeyPair solana.PrivateKey `koanf:"-"`
}

// Load loads the identity keypairs from files
func (i *Identities) Load() (err error) {
	if i.ActiveKeyPairFile != "" {
		i.ActiveKeyPair, err = solana.PrivateKeyFromSolanaKeygenFile(i.ActiveKeyPairFile)
		if err != nil {
			return fmt.Errorf("failed to load active keypair from %s: %w", i.ActiveKeyPairFile, err)
		}
	}

	if i.PassiveKeyPairFile != "" {
		i.PassiveKeyPair, err = solana.PrivateKeyFromSolanaKeygenFile(i.PassiveKeyPairFile)
		if err != nil {
			return fmt.Errorf("failed to load passive keypair from %s: %w", i.PassiveKeyPairFile, err)
		}
	}

	return nil
}

// Validate validates the validator configuration
func (v *Validator) Validate() error {
	// Validate client
	normalizedClient := constants.NormalizeClientName(v.Client)
	err := constants.ValidateClientName(normalizedClient)
	if err != nil {
		return err
	}
	v.Client = normalizedClient

	// Validate RPC URL
	_, err = url.Parse(v.RPCURL)
	if err != nil {
		return fmt.Errorf("validator.rpc_url %s is not a valid URL: %w", v.RPCURL, err)
	}

	activeSources := 0
	for _, source := range []string{v.VoteAccountPublicKey, v.Identities.ActivePublicKey, v.Identities.ActiveKeyPairFile} {
		if source != "" {
			activeSources++
		}
	}
	if activeSources != 1 {
		return fmt.Errorf("exactly one of validator.vote_account_pubkey, validator.identities.active_pubkey, or validator.identities.active must be configured")
	}

	if v.Identities.PassivePublicKey != "" && v.Identities.PassiveKeyPairFile != "" {
		return fmt.Errorf("only one of validator.identities.passive_pubkey or validator.identities.passive may be configured")
	}

	for name, publicKey := range map[string]string{
		"validator.vote_account_pubkey":       v.VoteAccountPublicKey,
		"validator.identities.active_pubkey":  v.Identities.ActivePublicKey,
		"validator.identities.passive_pubkey": v.Identities.PassivePublicKey,
	} {
		if publicKey == "" {
			continue
		}
		if _, err := solana.PublicKeyFromBase58(publicKey); err != nil {
			return fmt.Errorf("%s is not a valid Solana public key: %w", name, err)
		}
	}

	activePublicKey := v.Identities.ActiveIdentityPublicKey()
	passivePublicKey := v.Identities.PassiveIdentityPublicKey()
	if activePublicKey != "" && activePublicKey == passivePublicKey {
		return fmt.Errorf("active and passive validator identities must be different")
	}

	return nil
}

// ActiveIdentityPublicKey returns the configured or loaded active identity public key.
func (i *Identities) ActiveIdentityPublicKey() string {
	if i.ActivePublicKey != "" {
		return i.ActivePublicKey
	}
	if i.ActiveKeyPair != nil {
		return i.ActiveKeyPair.PublicKey().String()
	}
	return ""
}

// PassiveIdentityPublicKey returns the configured or loaded passive identity public key.
func (i *Identities) PassiveIdentityPublicKey() string {
	if i.PassivePublicKey != "" {
		return i.PassivePublicKey
	}
	if i.PassiveKeyPair != nil {
		return i.PassiveKeyPair.PublicKey().String()
	}
	return ""
}
