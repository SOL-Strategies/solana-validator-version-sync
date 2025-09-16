package validator

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func TestState_StructFields(t *testing.T) {
	v1, _ := version.NewVersion("1.18.0")
	state := State{
		Cluster:           "mainnet-beta",
		VersionString:     "1.18.0",
		HealthStatus:      "ok",
		IdentityPublicKey: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
		Version:           v1,
		Hostname:          "validator-01",
	}

	if state.Cluster != "mainnet-beta" {
		t.Errorf("Expected Cluster to be mainnet-beta, got %s", state.Cluster)
	}
	if state.VersionString != "1.18.0" {
		t.Errorf("Expected VersionString to be 1.18.0, got %s", state.VersionString)
	}
	if state.HealthStatus != "ok" {
		t.Errorf("Expected HealthStatus to be ok, got %s", state.HealthStatus)
	}
	if state.IdentityPublicKey != "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM" {
		t.Errorf("Expected IdentityPublicKey to be 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM, got %s", state.IdentityPublicKey)
	}
	if state.Version.String() != "1.18.0" {
		t.Errorf("Expected Version to be 1.18.0, got %s", state.Version.String())
	}
	if state.Hostname != "validator-01" {
		t.Errorf("Expected Hostname to be validator-01, got %s", state.Hostname)
	}
}

func TestState_EmptyState(t *testing.T) {
	state := State{}

	if state.Cluster != "" {
		t.Errorf("Expected empty Cluster, got %s", state.Cluster)
	}
	if state.VersionString != "" {
		t.Errorf("Expected empty VersionString, got %s", state.VersionString)
	}
	if state.HealthStatus != "" {
		t.Errorf("Expected empty HealthStatus, got %s", state.HealthStatus)
	}
	if state.IdentityPublicKey != "" {
		t.Errorf("Expected empty IdentityPublicKey, got %s", state.IdentityPublicKey)
	}
	if state.Version != nil {
		t.Errorf("Expected nil Version, got %v", state.Version)
	}
	if state.Hostname != "" {
		t.Errorf("Expected empty Hostname, got %s", state.Hostname)
	}
}

func TestState_VersionParsing(t *testing.T) {
	tests := []struct {
		name           string
		versionString  string
		expectedString string
		shouldParse    bool
	}{
		{
			name:           "valid semantic version",
			versionString:  "1.18.0",
			expectedString: "1.18.0",
			shouldParse:    true,
		},
		{
			name:           "valid semantic version with pre-release",
			versionString:  "1.18.0-beta.1",
			expectedString: "1.18.0-beta.1",
			shouldParse:    true,
		},
		{
			name:           "valid semantic version with build metadata",
			versionString:  "1.18.0+build.1",
			expectedString: "1.18.0+build.1",
			shouldParse:    true,
		},
		{
			name:           "invalid version string",
			versionString:  "invalid-version",
			expectedString: "",
			shouldParse:    false,
		},
		{
			name:           "empty version string",
			versionString:  "",
			expectedString: "",
			shouldParse:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				VersionString: tt.versionString,
			}

			if tt.shouldParse {
				parsedVersion, err := version.NewVersion(tt.versionString)
				if err != nil {
					t.Errorf("Failed to parse version %s: %v", tt.versionString, err)
				}
				state.Version = parsedVersion

				if state.Version.String() != tt.expectedString {
					t.Errorf("Expected version %s, got %s", tt.expectedString, state.Version.String())
				}
			} else {
				_, err := version.NewVersion(tt.versionString)
				if err == nil {
					t.Errorf("Expected error parsing invalid version %s", tt.versionString)
				}
			}
		})
	}
}

func TestState_HealthStatusValues(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "healthy status",
			status:   "ok",
			expected: "ok",
		},
		{
			name:     "unhealthy status",
			status:   "error",
			expected: "error",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			expected: "unknown",
		},
		{
			name:     "empty status",
			status:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				HealthStatus: tt.status,
			}

			if state.HealthStatus != tt.expected {
				t.Errorf("Expected HealthStatus %s, got %s", tt.expected, state.HealthStatus)
			}
		})
	}
}

func TestState_IdentityPublicKeyFormats(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "valid base58 public key",
			key:      "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			expected: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "",
		},
		{
			name:     "short key",
			key:      "short",
			expected: "short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				IdentityPublicKey: tt.key,
			}

			if state.IdentityPublicKey != tt.expected {
				t.Errorf("Expected IdentityPublicKey %s, got %s", tt.expected, state.IdentityPublicKey)
			}
		})
	}
}

func TestState_ClusterNames(t *testing.T) {
	tests := []struct {
		name     string
		cluster  string
		expected string
	}{
		{
			name:     "mainnet-beta",
			cluster:  "mainnet-beta",
			expected: "mainnet-beta",
		},
		{
			name:     "testnet",
			cluster:  "testnet",
			expected: "testnet",
		},
		{
			name:     "devnet",
			cluster:  "devnet",
			expected: "devnet",
		},
		{
			name:     "custom cluster",
			cluster:  "custom-cluster",
			expected: "custom-cluster",
		},
		{
			name:     "empty cluster",
			cluster:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				Cluster: tt.cluster,
			}

			if state.Cluster != tt.expected {
				t.Errorf("Expected Cluster %s, got %s", tt.expected, state.Cluster)
			}
		})
	}
}

func TestState_HostnameFormats(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected string
	}{
		{
			name:     "simple hostname",
			hostname: "validator-01",
			expected: "validator-01",
		},
		{
			name:     "hostname with domain",
			hostname: "validator-01.example.com",
			expected: "validator-01.example.com",
		},
		{
			name:     "IP address",
			hostname: "192.168.1.100",
			expected: "192.168.1.100",
		},
		{
			name:     "empty hostname",
			hostname: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				Hostname: tt.hostname,
			}

			if state.Hostname != tt.expected {
				t.Errorf("Expected Hostname %s, got %s", tt.expected, state.Hostname)
			}
		})
	}
}
