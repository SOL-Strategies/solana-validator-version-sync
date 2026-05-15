package github

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	goversion "github.com/hashicorp/go-version"
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
		constants.ClientNameRakurai,
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

func TestClientRepoConfigs_RakuraiConfig(t *testing.T) {
	config := clientRepoConfigs[constants.ClientNameRakurai]

	expectedURL := "https://github.com/rakurai-io/rakurai-validator"
	if config.URL != expectedURL {
		t.Errorf("Rakurai URL = %v, want %v", config.URL, expectedURL)
	}

	expectedClusters := []string{constants.ClusterNameMainnetBeta, constants.ClusterNameTestnet}
	for _, cluster := range expectedClusters {
		if _, exists := config.TagRegexes[cluster]; !exists {
			t.Errorf("Rakurai TagRegex not found for cluster: %s", cluster)
		}
	}

	if config.ReleaseNotesRegexes != nil {
		t.Errorf("Rakurai should not have ReleaseNotesRegexes, but found: %v", config.ReleaseNotesRegexes)
	}
	if config.ReleaseTitleRegexes != nil {
		t.Errorf("Rakurai should not have ReleaseTitleRegexes, but found: %v", config.ReleaseTitleRegexes)
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
			regex:      ".*(This is a stable release suitable for use on Mainnet Beta|This (?:is )?a stable Mainnet release|This (?:is )?(?:a )?Mainnet-beta Upgrade Candidate release).*",
		},
		{
			clientName: constants.ClientNameAgave,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseNotesRegex",
			regex:      "(?is).*(This is a testnet release|recommended for testnet).*",
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "ReleaseTitleRegex",
			regex:      "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?)-jito(?:\\.[0-9]+)?$",
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseTitleRegex",
			regex:      "^Testnet - v([0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?)-jito(?:\\.[0-9]+)?$",
		},
		{
			clientName: constants.ClientNameRakurai,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "TagRegex",
			regex:      "^release/(v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?-rakurai\\.[0-9]+)$",
		},
		{
			clientName: constants.ClientNameRakurai,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "TagRegex",
			regex:      "^release/(v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?-rakurai\\.[0-9]+)_testnet$",
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
			} else if tt.regexType == "ReleaseTitleRegex" {
				actualRegex, exists = config.ReleaseTitleRegexes[tt.cluster]
			} else {
				actualRegex, exists = config.TagRegexes[tt.cluster]
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

func TestNormalizeToTagVersion(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	tests := []struct {
		name           string
		clientName     string
		cachedVersions []string
		input          string
		want           string
	}{
		// Feature-set (PATCH) matching: post-PR #8945 RPC format where MINOR differs from tag
		{
			name:           "firedancer: normalizes 0.33670.40002 to 0.902.40002 by feature-set match",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.902.40002"},
			input:          "0.33670.40002",
			want:           "0.902.40002",
		},
		{
			name:           "firedancer: picks correct tag from multiple cached versions via feature-set",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.901.40001", "v0.902.40002", "v0.903.40003"},
			input:          "0.33670.40002",
			want:           "0.902.40002",
		},
		{
			name:           "firedancer: exact tag-format match wins over duplicate feature-set",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.821.30114", "v0.822.30114", "v0.909.40001"},
			input:          "0.822.30114",
			want:           "0.822.30114",
		},
		{
			name:           "firedancer: duplicate feature-set match picks latest tag",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.821.30114", "v0.822.30114"},
			input:          "0.33670.30114",
			want:           "0.822.30114",
		},
		// MAJOR.MINOR matching: current RPC format where PATCH is 0 (e.g. 0.902.0 vs tag v0.902.40002)
		{
			name:           "firedancer: normalizes 0.902.0 to 0.902.40002 by major.minor match",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.902.40002"},
			input:          "0.902.0",
			want:           "0.902.40002",
		},
		{
			name:           "firedancer: picks correct tag from multiple cached versions via major.minor",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.901.40001", "v0.902.40002", "v0.903.40003"},
			input:          "0.902.0",
			want:           "0.902.40002",
		},
		// Firedancer fallback: no match
		{
			name:           "firedancer: returns unchanged when cache is empty",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{},
			input:          "0.33670.40002",
			want:           "0.33670.40002",
		},
		{
			name:           "firedancer: returns unchanged when no cached tag matches either strategy",
			clientName:     constants.ClientNameFiredancer,
			cachedVersions: []string{"v0.901.40001"},
			input:          "0.33670.40002",
			want:           "0.33670.40002",
		},
		// Jito-Solana: RPC omits -jito suffix present in git tags
		{
			name:           "jito-solana: normalizes 4.0.0-beta.2 to v4.0.0-beta.2-jito by stripping -jito suffix",
			clientName:     constants.ClientNameJitoSolana,
			cachedVersions: []string{"v4.0.0-beta.2-jito"},
			input:          "4.0.0-beta.2",
			want:           "4.0.0-beta.2-jito",
		},
		{
			name:           "jito-solana: normalizes stable release 3.1.10 to v3.1.10-jito",
			clientName:     constants.ClientNameJitoSolana,
			cachedVersions: []string{"v3.1.10-jito"},
			input:          "3.1.10",
			want:           "3.1.10-jito",
		},
		{
			name:           "jito-solana: normalizes with -jito.N patch suffix in tag",
			clientName:     constants.ClientNameJitoSolana,
			cachedVersions: []string{"v3.0.6-jito.1"},
			input:          "3.0.6",
			want:           "3.0.6-jito.1",
		},
		{
			name:           "jito-solana: picks correct tag from multiple cached versions",
			clientName:     constants.ClientNameJitoSolana,
			cachedVersions: []string{"v3.1.9-jito", "v3.1.10-jito", "v4.0.0-beta.2-jito"},
			input:          "3.1.10",
			want:           "3.1.10-jito",
		},
		{
			name:           "jito-solana: returns unchanged when no cached tag matches",
			clientName:     constants.ClientNameJitoSolana,
			cachedVersions: []string{"v3.1.10-jito"},
			input:          "4.0.0-beta.2",
			want:           "4.0.0-beta.2",
		},
		// Agave: RPC version matches tag directly (no client suffix)
		{
			name:           "agave: returns matching cached tag for pre-release version",
			clientName:     constants.ClientNameAgave,
			cachedVersions: []string{"v2.2.8-beta.1"},
			input:          "2.2.8-beta.1",
			want:           "2.2.8-beta.1",
		},
		{
			name:           "agave: returns matching cached tag for stable version",
			clientName:     constants.ClientNameAgave,
			cachedVersions: []string{"v2.2.8"},
			input:          "2.2.8",
			want:           "2.2.8",
		},
		{
			name:           "agave: returns unchanged when no cached tag matches",
			clientName:     constants.ClientNameAgave,
			cachedVersions: []string{"v2.2.7"},
			input:          "2.2.8",
			want:           "2.2.8",
		},
		{
			name:           "rakurai: returns matching cached tag for stable version",
			clientName:     constants.ClientNameRakurai,
			cachedVersions: []string{"v3.1.8-rakurai.0"},
			input:          "3.1.8-rakurai.0",
			want:           "3.1.8-rakurai.0",
		},
		{
			name:           "rakurai: normalizes core version to matching tag version",
			clientName:     constants.ClientNameRakurai,
			cachedVersions: []string{"v3.1.8-rakurai.0"},
			input:          "3.1.8",
			want:           "3.1.8-rakurai.0",
		},
		{
			name:           "rakurai: returns unchanged when no cached tag matches",
			clientName:     constants.ClientNameRakurai,
			cachedVersions: []string{"v3.1.7-rakurai.0"},
			input:          "3.1.8",
			want:           "3.1.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := make([]*goversion.Version, 0, len(tt.cachedVersions))
			for _, s := range tt.cachedVersions {
				cached = append(cached, mustVersion(s))
			}

			c := &Client{
				clientName:        tt.clientName,
				cachedTagVersions: cached,
				logger:            log.WithPrefix("test"),
			}

			got := c.NormalizeToTagVersion(mustVersion(tt.input))
			if got.Core().String() != mustVersion(tt.want).Core().String() {
				t.Errorf("NormalizeToTagVersion(%q) = %q, want %q", tt.input, got.Core().String(), tt.want)
			}
		})
	}
}

func TestSelectRakuraiTagVersionInfo(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	tests := []struct {
		name        string
		cluster     string
		mainnetTags []tagVersionInfo
		testnetTags []tagVersionInfo
		want        string
		wantTag     string
		wantErr     bool
	}{
		{
			name:    "mainnet picks highest shared tag",
			cluster: constants.ClusterNameMainnetBeta,
			mainnetTags: []tagVersionInfo{
				{TagName: "release/v3.0.13-rakurai.0", Version: mustVersion("v3.0.13-rakurai.0")},
				{TagName: "release/v3.1.8-rakurai.0", Version: mustVersion("v3.1.8-rakurai.0")},
			},
			testnetTags: []tagVersionInfo{
				{TagName: "release/v3.1.8-rakurai.0_testnet", Version: mustVersion("v3.1.8-rakurai.0"), TestnetOnly: true},
			},
			want:    "3.1.8",
			wantTag: "release/v3.1.8-rakurai.0",
		},
		{
			name:    "testnet picks higher shared tag over lower testnet-only tag",
			cluster: constants.ClusterNameTestnet,
			mainnetTags: []tagVersionInfo{
				{TagName: "release/v3.1.8-rakurai.0", Version: mustVersion("v3.1.8-rakurai.0")},
			},
			testnetTags: []tagVersionInfo{
				{TagName: "release/v3.1.6-rakurai.0_testnet", Version: mustVersion("v3.1.6-rakurai.0"), TestnetOnly: true},
			},
			want:    "3.1.8",
			wantTag: "release/v3.1.8-rakurai.0",
		},
		{
			name:    "testnet prefers explicit testnet tag when equal version exists",
			cluster: constants.ClusterNameTestnet,
			mainnetTags: []tagVersionInfo{
				{TagName: "release/v3.1.8-rakurai.0", Version: mustVersion("v3.1.8-rakurai.0")},
			},
			testnetTags: []tagVersionInfo{
				{TagName: "release/v3.1.8-rakurai.0_testnet", Version: mustVersion("v3.1.8-rakurai.0"), TestnetOnly: true},
			},
			want:    "3.1.8",
			wantTag: "release/v3.1.8-rakurai.0_testnet",
		},
		{
			name:    "testnet falls back to shared tag when no testnet-only tag exists",
			cluster: constants.ClusterNameTestnet,
			mainnetTags: []tagVersionInfo{
				{TagName: "release/v3.1.8-rakurai.0", Version: mustVersion("v3.1.8-rakurai.0")},
			},
			want:    "3.1.8",
			wantTag: "release/v3.1.8-rakurai.0",
		},
		{
			name:    "mainnet errors when no eligible shared tag exists",
			cluster: constants.ClusterNameMainnetBeta,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				clientName: constants.ClientNameRakurai,
				cluster:    tt.cluster,
				logger:     log.WithPrefix("test"),
			}

			got, err := c.selectRakuraiTagVersionInfo(tt.mainnetTags, tt.testnetTags)
			if (err != nil) != tt.wantErr {
				t.Fatalf("selectRakuraiTagVersionInfo() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if got.Version == nil {
				t.Fatal("selectRakuraiTagVersionInfo() returned nil version")
			}
			if got.Version.Core().String() != tt.want {
				t.Errorf("selectRakuraiTagVersionInfo() version = %q, want %q", got.Version.Core().String(), tt.want)
			}
			if got.TagName != tt.wantTag {
				t.Errorf("selectRakuraiTagVersionInfo() tag = %q, want %q", got.TagName, tt.wantTag)
			}
		})
	}
}

func TestClientRepoConfigs_RakuraiReleaseTagRegex(t *testing.T) {
	config := clientRepoConfigs[constants.ClientNameRakurai]

	tests := []struct {
		name            string
		cluster         string
		tagName         string
		shouldMatch     bool
		expectedVersion string
	}{
		{
			name:            "mainnet/shared release tag",
			cluster:         constants.ClusterNameMainnetBeta,
			tagName:         "release/v3.1.8-rakurai.0",
			shouldMatch:     true,
			expectedVersion: "v3.1.8-rakurai.0",
		},
		{
			name:            "testnet-only release tag",
			cluster:         constants.ClusterNameTestnet,
			tagName:         "release/v3.1.8-rakurai.0_testnet",
			shouldMatch:     true,
			expectedVersion: "v3.1.8-rakurai.0",
		},
		{
			name:        "mainnet ignores testnet-only tag",
			cluster:     constants.ClusterNameMainnetBeta,
			tagName:     "release/v3.1.8-rakurai.0_testnet",
			shouldMatch: false,
		},
		{
			name:        "testnet ignores beta-like .b variant",
			cluster:     constants.ClusterNameTestnet,
			tagName:     "release/v3.0.14-rakurai.1.b_testnet",
			shouldMatch: false,
		},
		{
			name:        "mainnet ignores beta-like .b variant",
			cluster:     constants.ClusterNameMainnetBeta,
			tagName:     "release/v3.0.14-rakurai.1.b",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexStr, exists := config.TagRegexes[tt.cluster]
			if !exists {
				t.Fatalf("TagRegex not found for cluster: %s", tt.cluster)
			}

			re := regexp.MustCompile(regexStr)
			matches := re.FindStringSubmatch(tt.tagName)

			if tt.shouldMatch {
				if matches == nil {
					t.Fatalf("Expected regex to match %q, but it didn't", tt.tagName)
				}
				if matches[1] != tt.expectedVersion {
					t.Errorf("Expected version %q, got %q", tt.expectedVersion, matches[1])
				}
				return
			}

			if matches != nil {
				t.Errorf("Expected regex to NOT match %q, but it did (matched: %v)", tt.tagName, matches)
			}
		})
	}
}

func TestTagNameForVersion_Rakurai(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	testnetClient := &Client{
		clientName: constants.ClientNameRakurai,
		cluster:    constants.ClusterNameTestnet,
		cachedTagInfos: []tagVersionInfo{
			{TagName: "release/v3.1.8-rakurai.0", Version: mustVersion("v3.1.8-rakurai.0")},
			{TagName: "release/v3.1.8-rakurai.0_testnet", Version: mustVersion("v3.1.8-rakurai.0"), TestnetOnly: true},
		},
	}

	if got := testnetClient.TagNameForVersion(mustVersion("3.1.8")); got != "release/v3.1.8-rakurai.0_testnet" {
		t.Errorf("TagNameForVersion() testnet = %q, want %q", got, "release/v3.1.8-rakurai.0_testnet")
	}

	mainnetClient := &Client{
		clientName: constants.ClientNameRakurai,
		cluster:    constants.ClusterNameMainnetBeta,
		cachedTagInfos: []tagVersionInfo{
			{TagName: "release/v3.1.8-rakurai.0", Version: mustVersion("v3.1.8-rakurai.0")},
			{TagName: "release/v3.1.8-rakurai.0_testnet", Version: mustVersion("v3.1.8-rakurai.0"), TestnetOnly: true},
		},
	}

	if got := mainnetClient.TagNameForVersion(mustVersion("3.1.8")); got != "release/v3.1.8-rakurai.0" {
		t.Errorf("TagNameForVersion() mainnet = %q, want %q", got, "release/v3.1.8-rakurai.0")
	}
}

func TestClientRepoConfigs_JitoSolanaReleaseTitleRegex(t *testing.T) {
	config := clientRepoConfigs[constants.ClientNameJitoSolana]

	tests := []struct {
		name            string
		cluster         string
		releaseTitle    string
		shouldMatch     bool
		expectedVersion string
	}{
		{
			name:            "Mainnet stable",
			cluster:         constants.ClusterNameMainnetBeta,
			releaseTitle:    "Mainnet - v3.1.10-jito",
			shouldMatch:     true,
			expectedVersion: "3.1.10",
		},
		{
			name:            "Testnet stable",
			cluster:         constants.ClusterNameTestnet,
			releaseTitle:    "Testnet - v3.1.7-jito",
			shouldMatch:     true,
			expectedVersion: "3.1.7",
		},
		{
			name:            "Testnet pre-release beta",
			cluster:         constants.ClusterNameTestnet,
			releaseTitle:    "Testnet - v4.0.0-beta.2-jito",
			shouldMatch:     true,
			expectedVersion: "4.0.0-beta.2",
		},
		{
			name:            "Mainnet jito.N patch suffix",
			cluster:         constants.ClusterNameMainnetBeta,
			releaseTitle:    "Mainnet - v3.0.6-jito.1",
			shouldMatch:     true,
			expectedVersion: "3.0.6",
		},
		{
			name:         "Mainnet missing jito suffix",
			cluster:      constants.ClusterNameMainnetBeta,
			releaseTitle: "Mainnet - v3.1.10",
			shouldMatch:  false,
		},
		{
			name:         "Wrong network prefix",
			cluster:      constants.ClusterNameMainnetBeta,
			releaseTitle: "Testnet - v3.1.10-jito",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexStr, exists := config.ReleaseTitleRegexes[tt.cluster]
			if !exists {
				t.Fatalf("ReleaseTitleRegex not found for cluster: %s", tt.cluster)
			}

			re := regexp.MustCompile(regexStr)
			matches := re.FindStringSubmatch(tt.releaseTitle)

			if tt.shouldMatch {
				if matches == nil {
					t.Errorf("Expected regex to match %q, but it didn't", tt.releaseTitle)
					return
				}
				if matches[1] != tt.expectedVersion {
					t.Errorf("Expected version %q, got %q", tt.expectedVersion, matches[1])
				}
			} else {
				if matches != nil {
					t.Errorf("Expected regex to NOT match %q, but it did (matched: %v)", tt.releaseTitle, matches)
				}
			}
		})
	}
}
