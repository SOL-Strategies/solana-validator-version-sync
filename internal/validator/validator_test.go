package validator

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sync_commands"
)

func TestRoleConstants(t *testing.T) {
	if RoleActive != "active" {
		t.Errorf("Expected RoleActive to be 'active', got %s", RoleActive)
	}
	if RolePassive != "passive" {
		t.Errorf("Expected RolePassive to be 'passive', got %s", RolePassive)
	}
	if RoleUnknown != "unknown" {
		t.Errorf("Expected RoleUnknown to be 'unknown', got %s", RoleUnknown)
	}
}

func TestOptions_StructFields(t *testing.T) {
	opts := Options{
		Cluster: "mainnet-beta",
		SyncConfig: config.Sync{
			EnableSFDPCompliance: true,
		},
		ValidatorConfig: config.Validator{
			Client: "agave",
			RPCURL: "http://localhost:8899",
		},
	}

	if opts.Cluster != "mainnet-beta" {
		t.Errorf("Expected Cluster to be mainnet-beta, got %s", opts.Cluster)
	}
	if opts.SyncConfig.EnableSFDPCompliance != true {
		t.Errorf("Expected EnableSFDPCompliance to be true, got %v", opts.SyncConfig.EnableSFDPCompliance)
	}
	if opts.ValidatorConfig.Client != "agave" {
		t.Errorf("Expected Client to be agave, got %s", opts.ValidatorConfig.Client)
	}
	if opts.ValidatorConfig.RPCURL != "http://localhost:8899" {
		t.Errorf("Expected RPCURL to be http://localhost:8899, got %s", opts.ValidatorConfig.RPCURL)
	}
}

func TestValidator_StructFields(t *testing.T) {
	validator := Validator{
		ActiveIdentityPublicKey:  "active-key",
		PassiveIdentityPublicKey: "passive-key",
		State: State{
			Cluster: "mainnet-beta",
		},
	}

	if validator.ActiveIdentityPublicKey != "active-key" {
		t.Errorf("Expected ActiveIdentityPublicKey to be active-key, got %s", validator.ActiveIdentityPublicKey)
	}
	if validator.PassiveIdentityPublicKey != "passive-key" {
		t.Errorf("Expected PassiveIdentityPublicKey to be passive-key, got %s", validator.PassiveIdentityPublicKey)
	}
	if validator.State.Cluster != "mainnet-beta" {
		t.Errorf("Expected State.Cluster to be mainnet-beta, got %s", validator.State.Cluster)
	}
}

func TestValidator_Role(t *testing.T) {
	// Create test keypairs
	activeKeypair, _ := solana.NewRandomPrivateKey()
	passiveKeypair, _ := solana.NewRandomPrivateKey()
	unknownKeypair, _ := solana.NewRandomPrivateKey()

	tests := []struct {
		name                     string
		activeIdentityPublicKey  string
		passiveIdentityPublicKey string
		stateIdentityPublicKey   string
		expected                 string
	}{
		{
			name:                     "active role",
			activeIdentityPublicKey:  activeKeypair.PublicKey().String(),
			passiveIdentityPublicKey: passiveKeypair.PublicKey().String(),
			stateIdentityPublicKey:   activeKeypair.PublicKey().String(),
			expected:                 RoleActive,
		},
		{
			name:                     "passive role",
			activeIdentityPublicKey:  activeKeypair.PublicKey().String(),
			passiveIdentityPublicKey: passiveKeypair.PublicKey().String(),
			stateIdentityPublicKey:   passiveKeypair.PublicKey().String(),
			expected:                 RolePassive,
		},
		{
			name:                     "unknown role",
			activeIdentityPublicKey:  activeKeypair.PublicKey().String(),
			passiveIdentityPublicKey: passiveKeypair.PublicKey().String(),
			stateIdentityPublicKey:   unknownKeypair.PublicKey().String(),
			expected:                 RoleUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := Validator{
				ActiveIdentityPublicKey:  tt.activeIdentityPublicKey,
				PassiveIdentityPublicKey: tt.passiveIdentityPublicKey,
				State: State{
					IdentityPublicKey: tt.stateIdentityPublicKey,
				},
			}

			result := validator.Role()
			if result != tt.expected {
				t.Errorf("Role() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidator_IsActive(t *testing.T) {
	// Create test keypairs
	activeKeypair, _ := solana.NewRandomPrivateKey()
	passiveKeypair, _ := solana.NewRandomPrivateKey()

	validator := Validator{
		ActiveIdentityPublicKey:  activeKeypair.PublicKey().String(),
		PassiveIdentityPublicKey: passiveKeypair.PublicKey().String(),
		State: State{
			IdentityPublicKey: activeKeypair.PublicKey().String(),
		},
	}

	if !validator.IsActive() {
		t.Error("IsActive() should return true for active identity")
	}

	validator.State.IdentityPublicKey = passiveKeypair.PublicKey().String()
	if validator.IsActive() {
		t.Error("IsActive() should return false for passive identity")
	}
}

func TestValidator_IsPassive(t *testing.T) {
	// Create test keypairs
	activeKeypair, _ := solana.NewRandomPrivateKey()
	passiveKeypair, _ := solana.NewRandomPrivateKey()

	validator := Validator{
		ActiveIdentityPublicKey:  activeKeypair.PublicKey().String(),
		PassiveIdentityPublicKey: passiveKeypair.PublicKey().String(),
		State: State{
			IdentityPublicKey: passiveKeypair.PublicKey().String(),
		},
	}

	if !validator.IsPassive() {
		t.Error("IsPassive() should return true for passive identity")
	}

	validator.State.IdentityPublicKey = activeKeypair.PublicKey().String()
	if validator.IsPassive() {
		t.Error("IsPassive() should return false for active identity")
	}
}

func TestValidator_IsRoleUnknown(t *testing.T) {
	// Create test keypairs
	activeKeypair, _ := solana.NewRandomPrivateKey()
	passiveKeypair, _ := solana.NewRandomPrivateKey()
	unknownKeypair, _ := solana.NewRandomPrivateKey()

	validator := Validator{
		ActiveIdentityPublicKey:  activeKeypair.PublicKey().String(),
		PassiveIdentityPublicKey: passiveKeypair.PublicKey().String(),
		State: State{
			IdentityPublicKey: unknownKeypair.PublicKey().String(),
		},
	}

	if !validator.IsRoleUnknown() {
		t.Error("IsRoleUnknown() should return true for unknown identity")
	}

	validator.State.IdentityPublicKey = activeKeypair.PublicKey().String()
	if validator.IsRoleUnknown() {
		t.Error("IsRoleUnknown() should return false for active identity")
	}

	validator.State.IdentityPublicKey = passiveKeypair.PublicKey().String()
	if validator.IsRoleUnknown() {
		t.Error("IsRoleUnknown() should return false for passive identity")
	}
}

func TestNew(t *testing.T) {
	// Create test keypairs
	activeKeypair, _ := solana.NewRandomPrivateKey()
	passiveKeypair, _ := solana.NewRandomPrivateKey()

	opts := Options{
		Cluster: "mainnet-beta",
		SyncConfig: config.Sync{
			EnableSFDPCompliance: true,
			Commands: []sync_commands.Command{
				{
					Name: "test-command",
					Cmd:  "echo",
					Args: []string{"{{.VersionTo}}"},
				},
			},
		},
		ValidatorConfig: config.Validator{
			Client: constants.ClientNameAgave,
			RPCURL: "http://localhost:8899",
			Identities: config.Identities{
				ActiveKeyPair:  activeKeypair,
				PassiveKeyPair: passiveKeypair,
			},
		},
	}

	validator, err := New(opts)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if validator == nil {
		t.Error("New() returned nil validator")
	}

	if validator.ActiveIdentityPublicKey != activeKeypair.PublicKey().String() {
		t.Errorf("New() ActiveIdentityPublicKey = %v, want %v", validator.ActiveIdentityPublicKey, activeKeypair.PublicKey().String())
	}

	if validator.PassiveIdentityPublicKey != passiveKeypair.PublicKey().String() {
		t.Errorf("New() PassiveIdentityPublicKey = %v, want %v", validator.PassiveIdentityPublicKey, passiveKeypair.PublicKey().String())
	}

	if validator.State.Cluster != "mainnet-beta" {
		t.Errorf("New() State.Cluster = %v, want %v", validator.State.Cluster, "mainnet-beta")
	}

	if validator.syncConfig.EnableSFDPCompliance != true {
		t.Errorf("New() syncConfig.EnableSFDPCompliance = %v, want %v", validator.syncConfig.EnableSFDPCompliance, true)
	}

	if validator.cfg.Client != constants.ClientNameAgave {
		t.Errorf("New() cfg.Client = %v, want %v", validator.cfg.Client, constants.ClientNameAgave)
	}

	if validator.cfg.RPCURL != "http://localhost:8899" {
		t.Errorf("New() cfg.RPCURL = %v, want %v", validator.cfg.RPCURL, "http://localhost:8899")
	}

	if validator.logger == nil {
		t.Error("New() should set logger")
	}

	if validator.rpcClient == nil {
		t.Error("New() should set rpcClient")
	}

	if validator.sfdpClient == nil {
		t.Error("New() should set sfdpClient")
	}

	if validator.githubClient == nil {
		t.Error("New() should set githubClient")
	}
}

func TestNew_InvalidCommand(t *testing.T) {
	// Create test keypairs
	activeKeypair, _ := solana.NewRandomPrivateKey()
	passiveKeypair, _ := solana.NewRandomPrivateKey()

	opts := Options{
		Cluster: "mainnet-beta",
		SyncConfig: config.Sync{
			Commands: []sync_commands.Command{
				{
					Name: "invalid-command",
					Cmd:  "echo {{.InvalidTemplate", // Invalid template
				},
			},
		},
		ValidatorConfig: config.Validator{
			Client: constants.ClientNameAgave,
			RPCURL: "http://localhost:8899",
			Identities: config.Identities{
				ActiveKeyPair:  activeKeypair,
				PassiveKeyPair: passiveKeypair,
			},
		},
	}

	validator, err := New(opts)
	if err == nil {
		t.Error("New() should have failed with invalid command template")
	}

	if validator != nil {
		t.Error("New() should return nil validator on error")
	}
}
