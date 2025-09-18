package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestConfig_New(t *testing.T) {
	config, err := New()
	if err != nil {
		t.Errorf("New() error = %v, wantErr false", err)
	}
	if config == nil {
		t.Error("New() returned nil config")
	}
	if config.logger == nil {
		t.Error("New() should initialize logger")
	}
}

func TestConfig_LoadFromFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create temporary keypair files
	activeKeypair := solana.NewWallet()
	passiveKeypair := solana.NewWallet()

	activeKeyFile := filepath.Join(tempDir, "active-keypair.json")
	passiveKeyFile := filepath.Join(tempDir, "passive-keypair.json")

	err := writeKeypairFile(activeKeyFile, activeKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create active keypair file: %v", err)
	}

	err = writeKeypairFile(passiveKeyFile, passiveKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create passive keypair file: %v", err)
	}

	// Create a valid config file
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `log:
  level: debug
  format: json
validator:
  client: agave
  rpc_url: http://localhost:8899
  identities:
    active: ` + activeKeyFile + `
    passive: ` + passiveKeyFile + `
cluster:
  name: mainnet-beta
sync:
  enabled_when_active: true
  enable_sfdp_compliance: false
  allowed_semver_changes:
    major: false
    minor: true
    patch: true
  commands: []
`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	tests := []struct {
		name     string
		config   *Config
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid config file",
			config:   &Config{},
			filePath: configFile,
			wantErr:  false,
		},
		{
			name:     "non-existent config file",
			config:   &Config{},
			filePath: "/non/existent/config.yaml",
			wantErr:  true,
		},
		{
			name:     "invalid yaml content",
			config:   &Config{},
			filePath: createInvalidYamlFile(t, tempDir),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.LoadFromFile(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If loading was successful, verify the file path was set
			if !tt.wantErr && tt.config.File != tt.filePath {
				t.Errorf("Config.LoadFromFile() File = %v, want %v", tt.config.File, tt.filePath)
			}
		})
	}
}

func TestConfig_Initialize(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create temporary keypair files
	activeKeypair := solana.NewWallet()
	passiveKeypair := solana.NewWallet()

	activeKeyFile := filepath.Join(tempDir, "active-keypair.json")
	passiveKeyFile := filepath.Join(tempDir, "passive-keypair.json")

	err := writeKeypairFile(activeKeyFile, activeKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create active keypair file: %v", err)
	}

	err = writeKeypairFile(passiveKeyFile, passiveKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create passive keypair file: %v", err)
	}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &Config{
				Log: Log{
					Level:  "info",
					Format: "text",
				},
				Validator: Validator{
					Client: constants.ClientNameAgave,
					RPCURL: "http://localhost:8899",
					Identities: Identities{
						ActiveKeyPairFile:  activeKeyFile,
						PassiveKeyPairFile: passiveKeyFile,
					},
				},
				Cluster: Cluster{
					Name: constants.ClusterNameMainnetBeta,
				},
				Sync: Sync{
					EnabledWhenActive:    true,
					EnableSFDPCompliance: false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid log configuration",
			config: &Config{
				Log: Log{
					Level:  "invalid",
					Format: "text",
				},
				Validator: Validator{
					Client: constants.ClientNameAgave,
					RPCURL: "http://localhost:8899",
					Identities: Identities{
						ActiveKeyPairFile:  activeKeyFile,
						PassiveKeyPairFile: passiveKeyFile,
					},
				},
				Cluster: Cluster{
					Name: constants.ClusterNameMainnetBeta,
				},
				Sync: Sync{},
			},
			wantErr: true,
		},
		{
			name: "invalid validator configuration",
			config: &Config{
				Log: Log{
					Level:  "info",
					Format: "text",
				},
				Validator: Validator{
					Client: "invalid-client",
					RPCURL: "http://localhost:8899",
					Identities: Identities{
						ActiveKeyPairFile:  activeKeyFile,
						PassiveKeyPairFile: passiveKeyFile,
					},
				},
				Cluster: Cluster{
					Name: constants.ClusterNameMainnetBeta,
				},
				Sync: Sync{},
			},
			wantErr: true,
		},
		{
			name: "invalid cluster configuration",
			config: &Config{
				Log: Log{
					Level:  "info",
					Format: "text",
				},
				Validator: Validator{
					Client: constants.ClientNameAgave,
					RPCURL: "http://localhost:8899",
					Identities: Identities{
						ActiveKeyPairFile:  activeKeyFile,
						PassiveKeyPairFile: passiveKeyFile,
					},
				},
				Cluster: Cluster{
					Name: "invalid-cluster",
				},
				Sync: Sync{},
			},
			wantErr: true,
		},
		{
			name: "missing keypair files",
			config: &Config{
				Log: Log{
					Level:  "info",
					Format: "text",
				},
				Validator: Validator{
					Client: constants.ClientNameAgave,
					RPCURL: "http://localhost:8899",
					Identities: Identities{
						ActiveKeyPairFile:  "/non/existent/active.json",
						PassiveKeyPairFile: "/non/existent/passive.json",
					},
				},
				Cluster: Cluster{
					Name: constants.ClusterNameMainnetBeta,
				},
				Sync: Sync{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Initialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_NewFromConfigFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create temporary keypair files
	activeKeypair := solana.NewWallet()
	passiveKeypair := solana.NewWallet()

	activeKeyFile := filepath.Join(tempDir, "active-keypair.json")
	passiveKeyFile := filepath.Join(tempDir, "passive-keypair.json")

	err := writeKeypairFile(activeKeyFile, activeKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create active keypair file: %v", err)
	}

	err = writeKeypairFile(passiveKeyFile, passiveKeypair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create passive keypair file: %v", err)
	}

	// Create a valid config file
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `log:
  level: info
  format: text
validator:
  client: agave
  rpc_url: http://localhost:8899
  identities:
    active: ` + activeKeyFile + `
    passive: ` + passiveKeyFile + `
cluster:
  name: mainnet-beta
sync:
  enabled_when_active: true
  enable_sfdp_compliance: false
  allowed_semver_changes:
    major: false
    minor: true
    patch: true
  commands: []
`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid config file",
			filePath: configFile,
			wantErr:  false,
		},
		{
			name:     "non-existent config file",
			filePath: "/non/existent/config.yaml",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewFromConfigFile(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("NewFromConfigFile() returned nil config")
				} else if config.File != tt.filePath {
					t.Errorf("NewFromConfigFile() File = %v, want %v", config.File, tt.filePath)
				}
			}
		})
	}
}

func TestConfig_StructFields(t *testing.T) {
	config := &Config{
		Log: Log{
			Level:  "debug",
			Format: "json",
		},
		Validator: Validator{
			Client: constants.ClientNameJitoSolana,
			RPCURL: "https://api.mainnet-beta.solana.com",
		},
		Cluster: Cluster{
			Name: constants.ClusterNameTestnet,
		},
		Sync: Sync{
			EnabledWhenActive:    true,
			EnableSFDPCompliance: true,
		},
		File: "/path/to/config.yaml",
	}

	if config.Log.Level != "debug" {
		t.Errorf("Expected Log.Level to be debug, got %s", config.Log.Level)
	}
	if config.Validator.Client != constants.ClientNameJitoSolana {
		t.Errorf("Expected Validator.Client to be %s, got %s", constants.ClientNameJitoSolana, config.Validator.Client)
	}
	if config.Cluster.Name != constants.ClusterNameTestnet {
		t.Errorf("Expected Cluster.Name to be %s, got %s", constants.ClusterNameTestnet, config.Cluster.Name)
	}
	if config.Sync.EnabledWhenActive != true {
		t.Errorf("Expected Sync.EnabledWhenActive to be true, got %v", config.Sync.EnabledWhenActive)
	}
	if config.File != "/path/to/config.yaml" {
		t.Errorf("Expected File to be /path/to/config.yaml, got %s", config.File)
	}
}

// Helper function to create an invalid YAML file for testing
func createInvalidYamlFile(t *testing.T, tempDir string) string {
	filePath := filepath.Join(tempDir, "invalid.yaml")
	// Create invalid YAML content
	invalidContent := `log:
  level: debug
  format: json
validator:
  client: agave
  rpc_url: http://localhost:8899
  identities:
    active: /path/to/active.json
    passive: /path/to/passive.json
cluster:
  name: mainnet-beta
sync:
  enabled_when_active: true
  enable_sfdp_compliance: false
  allowed_semver_changes:
    major: false
    minor: true
    patch: true
  commands: []
invalid: [unclosed array
`
	err := os.WriteFile(filePath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}
	return filePath
}
