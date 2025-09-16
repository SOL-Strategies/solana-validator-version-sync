package github

import (
	"strings"
	"testing"

	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestClientRepoConfig_StructFields(t *testing.T) {
	config := ClientRepoConfig{
		URL: "https://github.com/test/repo",
		ReleaseNotesRegexes: map[string]string{
			"mainnet-beta": ".*mainnet.*",
			"testnet":      ".*testnet.*",
		},
		ReleaseTitleRegexes: map[string]string{
			"mainnet-beta": "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)$",
			"testnet":      "^Testnet - v([0-9]+\\.[0-9]+\\.[0-9]+)$",
		},
	}

	if config.URL != "https://github.com/test/repo" {
		t.Errorf("Expected URL to be https://github.com/test/repo, got %s", config.URL)
	}
	if len(config.ReleaseNotesRegexes) != 2 {
		t.Errorf("Expected ReleaseNotesRegexes to have 2 entries, got %d", len(config.ReleaseNotesRegexes))
	}
	if len(config.ReleaseTitleRegexes) != 2 {
		t.Errorf("Expected ReleaseTitleRegexes to have 2 entries, got %d", len(config.ReleaseTitleRegexes))
	}
}

func TestClientRepoConfigs_AllClients(t *testing.T) {
	expectedClients := []string{
		constants.ClientNameAgave,
		constants.ClientNameJitoSolana,
		constants.ClientNameFiredancer,
	}

	for _, clientName := range expectedClients {
		t.Run(clientName, func(t *testing.T) {
			config, exists := clientRepoConfigs[clientName]
			if !exists {
				t.Errorf("ClientRepoConfig not found for client: %s", clientName)
				return
			}

			// Verify URL is set
			if config.URL == "" {
				t.Errorf("ClientRepoConfig URL is empty for client: %s", clientName)
			}

			// Verify URL is a GitHub URL
			if !strings.Contains(config.URL, "github.com") {
				t.Errorf("ClientRepoConfig URL is not a GitHub URL for client: %s: %s", clientName, config.URL)
			}
		})
	}
}

func TestClientRepoConfigs_AgaveConfig(t *testing.T) {
	config := clientRepoConfigs[constants.ClientNameAgave]

	// Verify URL
	expectedURL := "https://github.com/anza-xyz/agave"
	if config.URL != expectedURL {
		t.Errorf("Agave URL = %v, want %v", config.URL, expectedURL)
	}

	// Verify ReleaseNotesRegexes exist for both clusters
	expectedClusters := []string{constants.ClusterNameMainnetBeta, constants.ClusterNameTestnet}
	for _, cluster := range expectedClusters {
		if _, exists := config.ReleaseNotesRegexes[cluster]; !exists {
			t.Errorf("Agave ReleaseNotesRegex not found for cluster: %s", cluster)
		}
	}

	// Agave should not have ReleaseTitleRegexes
	if config.ReleaseTitleRegexes != nil {
		t.Errorf("Agave should not have ReleaseTitleRegexes, but found: %v", config.ReleaseTitleRegexes)
	}
}

func TestClientRepoConfigs_JitoSolanaConfig(t *testing.T) {
	config := clientRepoConfigs[constants.ClientNameJitoSolana]

	// Verify URL
	expectedURL := "https://github.com/jito-foundation/jito-solana"
	if config.URL != expectedURL {
		t.Errorf("JitoSolana URL = %v, want %v", config.URL, expectedURL)
	}

	// Verify ReleaseTitleRegexes exist for both clusters
	expectedClusters := []string{constants.ClusterNameMainnetBeta, constants.ClusterNameTestnet}
	for _, cluster := range expectedClusters {
		if _, exists := config.ReleaseTitleRegexes[cluster]; !exists {
			t.Errorf("JitoSolana ReleaseTitleRegex not found for cluster: %s", cluster)
		}
	}

	// JitoSolana should not have ReleaseNotesRegexes
	if config.ReleaseNotesRegexes != nil {
		t.Errorf("JitoSolana should not have ReleaseNotesRegexes, but found: %v", config.ReleaseNotesRegexes)
	}
}

func TestClientRepoConfigs_FiredancerConfig(t *testing.T) {
	config := clientRepoConfigs[constants.ClientNameFiredancer]

	// Verify URL
	expectedURL := "https://github.com/firedancer-io/firedancer"
	if config.URL != expectedURL {
		t.Errorf("Firedancer URL = %v, want %v", config.URL, expectedURL)
	}

	// Verify ReleaseTitleRegexes exist for both clusters
	expectedClusters := []string{constants.ClusterNameMainnetBeta, constants.ClusterNameTestnet}
	for _, cluster := range expectedClusters {
		if _, exists := config.ReleaseTitleRegexes[cluster]; !exists {
			t.Errorf("Firedancer ReleaseTitleRegex not found for cluster: %s", cluster)
		}
	}

	// Firedancer should not have ReleaseNotesRegexes
	if config.ReleaseNotesRegexes != nil {
		t.Errorf("Firedancer should not have ReleaseNotesRegexes, but found: %v", config.ReleaseNotesRegexes)
	}
}

func TestClientRepoConfigs_RegexPatterns(t *testing.T) {
	tests := []struct {
		clientName string
		cluster    string
		regexType  string
		regex      string
	}{
		{
			clientName: constants.ClientNameAgave,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "ReleaseNotesRegex",
			regex:      ".*This is a stable release suitable for use on Mainnet Beta.*",
		},
		{
			clientName: constants.ClientNameAgave,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseNotesRegex",
			regex:      ".*This is a Testnet release.*",
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "ReleaseTitleRegex",
			regex:      "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito$",
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseTitleRegex",
			regex:      "^Testnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito$",
		},
		{
			clientName: constants.ClientNameFiredancer,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "ReleaseTitleRegex",
			regex:      "^(.*)dancer Mainnet v([0-9]+\\.[0-9]+\\.[0-9]+)$",
		},
		{
			clientName: constants.ClientNameFiredancer,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseTitleRegex",
			regex:      "^(.*)dancer Testnet v([0-9]+\\.[0-9]+\\.[0-9]+)$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.clientName+"_"+tt.cluster+"_"+tt.regexType, func(t *testing.T) {
			config := clientRepoConfigs[tt.clientName]

			var actualRegex string
			var exists bool

			if tt.regexType == "ReleaseNotesRegex" {
				actualRegex, exists = config.ReleaseNotesRegexes[tt.cluster]
			} else {
				actualRegex, exists = config.ReleaseTitleRegexes[tt.cluster]
			}

			if !exists {
				t.Errorf("%s not found for %s cluster", tt.regexType, tt.cluster)
				return
			}

			if actualRegex != tt.regex {
				t.Errorf("%s = %v, want %v", tt.regexType, actualRegex, tt.regex)
			}
		})
	}
}
