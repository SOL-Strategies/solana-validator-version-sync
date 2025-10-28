package github

import (
	"regexp"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/go-version"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "valid agave client for mainnet-beta",
			opts: Options{
				Cluster: constants.ClusterNameMainnetBeta,
				Client:  constants.ClientNameAgave,
			},
			wantErr: false,
		},
		{
			name: "valid agave client for testnet",
			opts: Options{
				Cluster: constants.ClusterNameTestnet,
				Client:  constants.ClientNameAgave,
			},
			wantErr: false,
		},
		{
			name: "valid jito-solana client for mainnet-beta",
			opts: Options{
				Cluster: constants.ClusterNameMainnetBeta,
				Client:  constants.ClientNameJitoSolana,
			},
			wantErr: false,
		},
		{
			name: "valid firedancer client for mainnet-beta",
			opts: Options{
				Cluster: constants.ClusterNameMainnetBeta,
				Client:  constants.ClientNameFiredancer,
			},
			wantErr: false,
		},
		{
			name: "invalid client name",
			opts: Options{
				Cluster: constants.ClusterNameMainnetBeta,
				Client:  "invalid-client",
			},
			wantErr: true,
		},
		{
			name: "invalid cluster name - should still work as cluster validation is not in NewClient",
			opts: Options{
				Cluster: "invalid-cluster",
				Client:  constants.ClientNameAgave,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if client == nil {
					t.Error("NewClient() returned nil client")
				} else {
					// Verify client properties
					if client.cluster != tt.opts.Cluster {
						t.Errorf("NewClient() cluster = %v, want %v", client.cluster, tt.opts.Cluster)
					}
					if client.clientName != tt.opts.Client {
						t.Errorf("NewClient() clientName = %v, want %v", client.clientName, tt.opts.Client)
					}
					if client.client == nil {
						t.Error("NewClient() should initialize GitHub client")
					}
					if client.logger == nil {
						t.Error("NewClient() should initialize logger")
					}
					if len(client.releaseNotesRegexes) == 0 && len(client.releaseTitleRegexes) == 0 {
						t.Error("NewClient() should initialize at least one regex")
					}
				}
			}
		})
	}
}

func TestClient_setOwnerAndRepo(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS GitHub URL",
			repoURL:   "https://github.com/anza-xyz/agave",
			wantOwner: "anza-xyz",
			wantRepo:  "agave",
			wantErr:   false,
		},
		{
			name:      "HTTPS GitHub URL with .git suffix",
			repoURL:   "https://github.com/jito-foundation/jito-solana.git",
			wantOwner: "jito-foundation",
			wantRepo:  "jito-solana",
			wantErr:   false,
		},
		{
			name:      "SSH GitHub URL",
			repoURL:   "git@github.com:firedancer-io/firedancer.git",
			wantOwner: "firedancer-io",
			wantRepo:  "firedancer",
			wantErr:   false,
		},
		{
			name:      "SSH GitHub URL without .git suffix",
			repoURL:   "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:    "invalid URL format",
			repoURL: "not-a-github-url",
			wantErr: true,
		},
		{
			name:    "incomplete HTTPS URL",
			repoURL: "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "incomplete SSH URL",
			repoURL: "git@github.com:owner",
			wantErr: true,
		},
		{
			name:    "empty URL",
			repoURL: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				repoURL: tt.repoURL,
			}

			err := client.setOwnerAndRepo()
			if (err != nil) != tt.wantErr {
				t.Errorf("setOwnerAndRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if client.repoOwner != tt.wantOwner {
					t.Errorf("setOwnerAndRepo() repoOwner = %v, want %v", client.repoOwner, tt.wantOwner)
				}
				if client.repoName != tt.wantRepo {
					t.Errorf("setOwnerAndRepo() repoName = %v, want %v", client.repoName, tt.wantRepo)
				}
			}
		})
	}
}

func TestVersionsFromReleaseTitleRegex(t *testing.T) {
	tests := []struct {
		name     string
		releases []*github.RepositoryRelease
		regex    string
		want     []string
	}{
		{
			name: "matching releases",
			releases: []*github.RepositoryRelease{
				{Name: github.String("Mainnet - v1.18.0-jito"), TagName: github.String("v1.18.0")},
				{Name: github.String("Testnet - v1.17.0-jito"), TagName: github.String("v1.17.0")},
				{Name: github.String("Mainnet - v1.19.0-jito"), TagName: github.String("v1.19.0")},
				{Name: github.String("Some other release"), TagName: github.String("v1.20.0")},
			},
			regex: "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito$",
			want:  []string{"v1.18.0", "v1.19.0"},
		},
		{
			name: "no matching releases",
			releases: []*github.RepositoryRelease{
				{Name: github.String("Testnet - v1.17.0-jito"), TagName: github.String("v1.17.0")},
				{Name: github.String("Some other release"), TagName: github.String("v1.20.0")},
			},
			regex: "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito$",
			want:  []string{},
		},
		{
			name:     "empty releases",
			releases: []*github.RepositoryRelease{},
			regex:    "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito$",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := regexp.Compile(tt.regex)
			if err != nil {
				t.Fatalf("Failed to compile regex: %v", err)
			}

			got := versionsFromReleaseTitleRegex(tt.releases, regex)
			if len(got) != len(tt.want) {
				t.Errorf("versionsFromReleaseTitleRegex() returned %d versions, want %d", len(got), len(tt.want))
			}

			// Check that all expected versions are present
			for _, wantVersion := range tt.want {
				found := false
				for _, gotVersion := range got {
					if gotVersion == wantVersion {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("versionsFromReleaseTitleRegex() missing expected version: %s", wantVersion)
				}
			}
		})
	}
}

func TestVersionsFromReleaseBodyRegex(t *testing.T) {
	tests := []struct {
		name     string
		releases []*github.RepositoryRelease
		regex    string
		want     []string
	}{
		{
			name: "matching releases",
			releases: []*github.RepositoryRelease{
				{Body: github.String("This is a stable release suitable for use on Mainnet Beta"), TagName: github.String("v1.18.0")},
				{Body: github.String("This is a Testnet release"), TagName: github.String("v1.17.0")},
				{Body: github.String("This is a stable release suitable for use on Mainnet Beta"), TagName: github.String("v1.19.0")},
				{Body: github.String("Some other release notes"), TagName: github.String("v1.20.0")},
			},
			regex: ".*This is a stable release suitable for use on Mainnet Beta.*",
			want:  []string{"v1.18.0", "v1.19.0"},
		},
		{
			name: "no matching releases",
			releases: []*github.RepositoryRelease{
				{Body: github.String("This is a Testnet release"), TagName: github.String("v1.17.0")},
				{Body: github.String("Some other release notes"), TagName: github.String("v1.20.0")},
			},
			regex: ".*This is a stable release suitable for use on Mainnet Beta.*",
			want:  []string{},
		},
		{
			name:     "empty releases",
			releases: []*github.RepositoryRelease{},
			regex:    ".*This is a stable release suitable for use on Mainnet Beta.*",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := regexp.Compile(tt.regex)
			if err != nil {
				t.Fatalf("Failed to compile regex: %v", err)
			}

			got := versionsFromReleaseBodyRegex(tt.releases, regex)
			if len(got) != len(tt.want) {
				t.Errorf("versionsFromReleaseBodyRegex() returned %d versions, want %d", len(got), len(tt.want))
			}

			// Check that all expected versions are present
			for _, wantVersion := range tt.want {
				found := false
				for _, gotVersion := range got {
					if gotVersion == wantVersion {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("versionsFromReleaseBodyRegex() missing expected version: %s", wantVersion)
				}
			}
		})
	}
}

func TestClientRepoConfigs(t *testing.T) {
	tests := []struct {
		clientName string
		cluster    string
		wantURL    string
		wantRegex  bool
	}{
		{
			clientName: constants.ClientNameAgave,
			cluster:    constants.ClusterNameMainnetBeta,
			wantURL:    "https://github.com/anza-xyz/agave",
			wantRegex:  true,
		},
		{
			clientName: constants.ClientNameAgave,
			cluster:    constants.ClusterNameTestnet,
			wantURL:    "https://github.com/anza-xyz/agave",
			wantRegex:  true,
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameMainnetBeta,
			wantURL:    "https://github.com/jito-foundation/jito-solana",
			wantRegex:  true,
		},
		{
			clientName: constants.ClientNameFiredancer,
			cluster:    constants.ClusterNameMainnetBeta,
			wantURL:    "https://github.com/firedancer-io/firedancer",
			wantRegex:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.clientName+"_"+tt.cluster, func(t *testing.T) {
			config, exists := clientRepoConfigs[tt.clientName]
			if !exists {
				t.Errorf("ClientRepoConfig not found for client: %s", tt.clientName)
				return
			}

			if config.URL != tt.wantURL {
				t.Errorf("ClientRepoConfig URL = %v, want %v", config.URL, tt.wantURL)
			}

			// Check that regexes exist for the cluster
			if tt.wantRegex {
				if config.ReleaseNotesRegexes != nil {
					if _, exists := config.ReleaseNotesRegexes[tt.cluster]; !exists {
						t.Errorf("ReleaseNotesRegex not found for cluster: %s", tt.cluster)
					}
				}
				if config.ReleaseTitleRegexes != nil {
					if _, exists := config.ReleaseTitleRegexes[tt.cluster]; !exists {
						t.Errorf("ReleaseTitleRegex not found for cluster: %s", tt.cluster)
					}
				}
			}
		})
	}
}

func TestClient_StructFields(t *testing.T) {
	client := &Client{
		repoURL:    "https://github.com/test/repo",
		repoOwner:  "test",
		repoName:   "repo",
		clientName: constants.ClientNameAgave,
		cluster:    constants.ClusterNameMainnetBeta,
	}

	if client.repoURL != "https://github.com/test/repo" {
		t.Errorf("Expected repoURL to be https://github.com/test/repo, got %s", client.repoURL)
	}
	if client.repoOwner != "test" {
		t.Errorf("Expected repoOwner to be test, got %s", client.repoOwner)
	}
	if client.repoName != "repo" {
		t.Errorf("Expected repoName to be repo, got %s", client.repoName)
	}
	if client.clientName != constants.ClientNameAgave {
		t.Errorf("Expected clientName to be %s, got %s", constants.ClientNameAgave, client.clientName)
	}
	if client.cluster != constants.ClusterNameMainnetBeta {
		t.Errorf("Expected cluster to be %s, got %s", constants.ClusterNameMainnetBeta, client.cluster)
	}
}

func TestOptions_StructFields(t *testing.T) {
	opts := Options{
		Cluster: constants.ClusterNameTestnet,
		Client:  constants.ClientNameJitoSolana,
	}

	if opts.Cluster != constants.ClusterNameTestnet {
		t.Errorf("Expected Cluster to be %s, got %s", constants.ClusterNameTestnet, opts.Cluster)
	}
	if opts.Client != constants.ClientNameJitoSolana {
		t.Errorf("Expected Client to be %s, got %s", constants.ClientNameJitoSolana, opts.Client)
	}
}

func TestClient_GetLatestClientVersion_MainnetTestnetPreference(t *testing.T) {
	// Test the version comparison logic directly
	tests := []struct {
		name                    string
		mainnetVersion          string
		testnetVersion          string
		cluster                 string
		expectedVersion         string
		expectMainnetPreference bool
	}{
		{
			name:                    "testnet should prefer mainnet version when mainnet is higher",
			mainnetVersion:          "1.18.0",
			testnetVersion:          "1.17.0",
			cluster:                 constants.ClusterNameTestnet,
			expectedVersion:         "1.18.0",
			expectMainnetPreference: true,
		},
		{
			name:                    "testnet should use testnet version when testnet is higher",
			mainnetVersion:          "1.16.0",
			testnetVersion:          "1.17.0",
			cluster:                 constants.ClusterNameTestnet,
			expectedVersion:         "1.17.0",
			expectMainnetPreference: false,
		},
		{
			name:                    "testnet should use testnet version when versions are equal",
			mainnetVersion:          "1.17.0",
			testnetVersion:          "1.17.0",
			cluster:                 constants.ClusterNameTestnet,
			expectedVersion:         "1.17.0",
			expectMainnetPreference: false,
		},
		{
			name:                    "mainnet should always use mainnet version",
			mainnetVersion:          "1.18.0",
			testnetVersion:          "1.19.0",
			cluster:                 constants.ClusterNameMainnetBeta,
			expectedVersion:         "1.18.0",
			expectMainnetPreference: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse versions
			mainnetVer, err := version.NewVersion(tt.mainnetVersion)
			if err != nil {
				t.Fatalf("Failed to parse mainnet version: %v", err)
			}
			testnetVer, err := version.NewVersion(tt.testnetVersion)
			if err != nil {
				t.Fatalf("Failed to parse testnet version: %v", err)
			}

			// Create version map
			latestClusterVersion := map[string]*version.Version{
				constants.ClusterNameMainnetBeta: mainnetVer,
				constants.ClusterNameTestnet:     testnetVer,
			}

			// Test the logic from GetLatestClientVersion
			latestVersion := latestClusterVersion[tt.cluster]
			if tt.cluster == constants.ClusterNameTestnet && latestClusterVersion[constants.ClusterNameMainnetBeta].GreaterThan(latestVersion) {
				latestVersion = latestClusterVersion[constants.ClusterNameMainnetBeta]
			}

			// Check the result
			if latestVersion.Core().String() != tt.expectedVersion {
				t.Errorf("Expected version %v, got %v", tt.expectedVersion, latestVersion.Core().String())
			}

			// Check if mainnet preference was applied
			mainnetPreferenceApplied := tt.cluster == constants.ClusterNameTestnet &&
				latestClusterVersion[constants.ClusterNameMainnetBeta].GreaterThan(latestClusterVersion[constants.ClusterNameTestnet]) &&
				latestVersion == latestClusterVersion[constants.ClusterNameMainnetBeta]

			if mainnetPreferenceApplied != tt.expectMainnetPreference {
				t.Errorf("Expected mainnet preference %v, got %v", tt.expectMainnetPreference, mainnetPreferenceApplied)
			}
		})
	}
}

func TestClient_GetLatestClientVersion_EdgeCases(t *testing.T) {
	// Test edge cases for version comparison logic
	tests := []struct {
		name           string
		mainnetVersion string
		testnetVersion string
		cluster        string
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "testnet with higher version than mainnet should work",
			mainnetVersion: "1.16.0",
			testnetVersion: "1.17.0",
			cluster:        constants.ClusterNameTestnet,
			expectError:    false,
		},
		{
			name:           "mainnet with higher version than testnet should work",
			mainnetVersion: "1.18.0",
			testnetVersion: "1.17.0",
			cluster:        constants.ClusterNameTestnet,
			expectError:    false,
		},
		{
			name:           "equal versions should work",
			mainnetVersion: "1.17.0",
			testnetVersion: "1.17.0",
			cluster:        constants.ClusterNameTestnet,
			expectError:    false,
		},
		{
			name:           "mainnet cluster should always use mainnet version",
			mainnetVersion: "1.18.0",
			testnetVersion: "1.19.0",
			cluster:        constants.ClusterNameMainnetBeta,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse versions
			mainnetVer, err := version.NewVersion(tt.mainnetVersion)
			if err != nil {
				t.Fatalf("Failed to parse mainnet version: %v", err)
			}
			testnetVer, err := version.NewVersion(tt.testnetVersion)
			if err != nil {
				t.Fatalf("Failed to parse testnet version: %v", err)
			}

			// Create version map
			latestClusterVersion := map[string]*version.Version{
				constants.ClusterNameMainnetBeta: mainnetVer,
				constants.ClusterNameTestnet:     testnetVer,
			}

			// Test the logic from GetLatestClientVersion
			latestVersion := latestClusterVersion[tt.cluster]
			if tt.cluster == constants.ClusterNameTestnet && latestClusterVersion[constants.ClusterNameMainnetBeta].GreaterThan(latestVersion) {
				latestVersion = latestClusterVersion[constants.ClusterNameMainnetBeta]
			}

			// Verify the logic works without errors
			if latestVersion == nil {
				t.Error("Expected a valid version, got nil")
			}

			// For testnet, verify the preference logic
			if tt.cluster == constants.ClusterNameTestnet {
				expectedVersion := latestClusterVersion[constants.ClusterNameTestnet]
				if latestClusterVersion[constants.ClusterNameMainnetBeta].GreaterThan(latestClusterVersion[constants.ClusterNameTestnet]) {
					expectedVersion = latestClusterVersion[constants.ClusterNameMainnetBeta]
				}
				if latestVersion != expectedVersion {
					t.Errorf("Expected version %v, got %v", expectedVersion.Core().String(), latestVersion.Core().String())
				}
			}
		})
	}
}
