package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		validator Validator
		wantErr   bool
	}{
		{
			name: "valid agave validator",
			validator: Validator{
				Client:            constants.ClientNameAgave,
				RPCURL:            "http://localhost:8899",
				VersionConstraint: ">= 1.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid jito-solana validator",
			validator: Validator{
				Client:            constants.ClientNameJitoSolana,
				RPCURL:            "https://api.mainnet-beta.solana.com",
				VersionConstraint: ">= 3.0.0, < 3.0.1",
			},
			wantErr: false,
		},
		{
			name: "valid firedancer validator",
			validator: Validator{
				Client:            constants.ClientNameFiredancer,
				RPCURL:            "http://127.0.0.1:8899",
				VersionConstraint: ">= 0.1.0",
			},
			wantErr: false,
		},
		{
			name: "invalid client name",
			validator: Validator{
				Client: "invalid-client",
				RPCURL: "http://localhost:8899",
			},
			wantErr: true,
		},
		{
			name: "invalid RPC URL - malformed scheme",
			validator: Validator{
				Client: constants.ClientNameAgave,
				RPCURL: "://invalid",
			},
			wantErr: true,
		},
		{
			name: "empty client name",
			validator: Validator{
				Client: "",
				RPCURL: "http://localhost:8899",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIdentities_Load(t *testing.T) {
	// Create temporary directory for test keypair files
	tempDir := t.TempDir()

	// Generate test keypairs
	activeKeypair := solana.NewWallet()
	passiveKeypair := solana.NewWallet()

	// Create temporary keypair files
	activeKeyFile := filepath.Join(tempDir, "active-keypair.json")
	passiveKeyFile := filepath.Join(tempDir, "passive-keypair.json")

	// Write keypair files in the solana keygen format (array of 64 bytes)
	err := writeKeypairFile(activeKeyFile, activeKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create active keypair file: %v", err)
	}

	err = writeKeypairFile(passiveKeyFile, passiveKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create passive keypair file: %v", err)
	}

	tests := []struct {
		name       string
		identities Identities
		wantErr    bool
	}{
		{
			name: "valid keypair files",
			identities: Identities{
				ActiveKeyPairFile:  activeKeyFile,
				PassiveKeyPairFile: passiveKeyFile,
			},
			wantErr: false,
		},
		{
			name: "non-existent active keypair file",
			identities: Identities{
				ActiveKeyPairFile:  "/non/existent/active.json",
				PassiveKeyPairFile: passiveKeyFile,
			},
			wantErr: true,
		},
		{
			name: "non-existent passive keypair file",
			identities: Identities{
				ActiveKeyPairFile:  activeKeyFile,
				PassiveKeyPairFile: "/non/existent/passive.json",
			},
			wantErr: true,
		},
		{
			name: "invalid keypair file content",
			identities: Identities{
				ActiveKeyPairFile:  createInvalidKeypairFile(t, tempDir, "invalid-active.json"),
				PassiveKeyPairFile: passiveKeyFile,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.identities.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Identities.Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If loading was successful, verify the keypairs were loaded
			if !tt.wantErr {
				if tt.identities.ActiveKeyPair == nil {
					t.Error("ActiveKeyPair should be loaded")
				}
				if tt.identities.PassiveKeyPair == nil {
					t.Error("PassiveKeyPair should be loaded")
				}
			}
		})
	}
}

func TestValidator_StructFields(t *testing.T) {
	validator := Validator{
		Client:            constants.ClientNameAgave,
		RPCURL:            "http://localhost:8899",
		VersionConstraint: ">= 1.0.0",
		Identities: Identities{
			ActiveKeyPairFile:  "/path/to/active.json",
			PassiveKeyPairFile: "/path/to/passive.json",
		},
	}

	if validator.Client != constants.ClientNameAgave {
		t.Errorf("Expected Client to be %s, got %s", constants.ClientNameAgave, validator.Client)
	}
	if validator.RPCURL != "http://localhost:8899" {
		t.Errorf("Expected RPCURL to be http://localhost:8899, got %s", validator.RPCURL)
	}
	if validator.VersionConstraint != ">= 1.0.0" {
		t.Errorf("Expected VersionConstraint to be >= 1.0.0, got %s", validator.VersionConstraint)
	}
	if validator.Identities.ActiveKeyPairFile != "/path/to/active.json" {
		t.Errorf("Expected ActiveKeyPairFile to be /path/to/active.json, got %s", validator.Identities.ActiveKeyPairFile)
	}
	if validator.Identities.PassiveKeyPairFile != "/path/to/passive.json" {
		t.Errorf("Expected PassiveKeyPairFile to be /path/to/passive.json, got %s", validator.Identities.PassiveKeyPairFile)
	}
}

func TestIdentities_StructFields(t *testing.T) {
	identities := Identities{
		ActiveKeyPairFile:  "/path/to/active.json",
		PassiveKeyPairFile: "/path/to/passive.json",
	}

	if identities.ActiveKeyPairFile != "/path/to/active.json" {
		t.Errorf("Expected ActiveKeyPairFile to be /path/to/active.json, got %s", identities.ActiveKeyPairFile)
	}
	if identities.PassiveKeyPairFile != "/path/to/passive.json" {
		t.Errorf("Expected PassiveKeyPairFile to be /path/to/passive.json, got %s", identities.PassiveKeyPairFile)
	}
	if identities.ActiveKeyPair != nil {
		t.Error("Expected ActiveKeyPair to be nil initially")
	}
	if identities.PassiveKeyPair != nil {
		t.Error("Expected PassiveKeyPair to be nil initially")
	}
}

// Helper function to write a keypair file in solana keygen format
func writeKeypairFile(filePath string, privateKey solana.PrivateKey) error {
	// Convert private key to byte array format expected by solana keygen files
	keyBytes := []byte(privateKey)

	// Write as JSON array of bytes
	jsonData, err := json.Marshal(keyBytes)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// Helper function to create an invalid keypair file for testing
func createInvalidKeypairFile(t *testing.T, tempDir, filename string) string {
	filePath := filepath.Join(tempDir, filename)
	err := os.WriteFile(filePath, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid keypair file: %v", err)
	}
	return filePath
}
