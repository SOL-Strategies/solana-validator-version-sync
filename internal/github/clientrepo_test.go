package github

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	gogithub "github.com/google/go-github/v74/github"
	goversion "github.com/hashicorp/go-version"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

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

	if _, exists := config.ReleaseNotesRegexes[constants.ClusterNameMainnetBeta]; !exists {
		t.Errorf("Firedancer ReleaseNotesRegex not found for cluster: %s", constants.ClusterNameMainnetBeta)
	}
	if _, exists := config.ReleaseNotesRegexes[constants.ClusterNameTestnet]; exists {
		t.Errorf("Firedancer should not need a testnet ReleaseNotesRegex, but found one")
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
			regex:      "(?i).*(This (?:is )?a stable release suitable for [^\\n]*Mainnet Beta|This (?:is )?a stable Mainnet release|This (?:is )?a stable release\\s*(?:[.\\r\\n]|$)|(?:This (?:is )?(?:a )?)?Mainnet(?:[- ]Beta)? Upgrade Candidate(?: release)?).*",
		},
		{
			clientName: constants.ClientNameAgave,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseNotesRegex",
			regex:      "(?is).*(This is a testnet release|recommended for testnet|suitable for testnet).*",
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "ReleaseTitleRegex",
			regex:      "^Mainnet\\s+-\\s+(?:Release\\s+)?v([0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?)-jito(?:\\.[0-9]+)?$",
		},
		{
			clientName: constants.ClientNameJitoSolana,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseTitleRegex",
			regex:      "^Testnet\\s+-\\s+(?:Release\\s+)?v([0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?)-jito(?:\\.[0-9]+)?$",
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
			regex:      "^(.*)dancer Mainnet v([0-9]+\\.[0-9]+\\.[0-9]+)(?:\\b.*)?$",
		},
		{
			clientName: constants.ClientNameFiredancer,
			cluster:    constants.ClusterNameMainnetBeta,
			regexType:  "ReleaseNotesRegex",
			regex:      "(?is).*This is a Testnet release\\.[^\\n]*(may also be used on mainnet|also (?:be )?suitable for mainnet).*",
		},
		{
			clientName: constants.ClientNameFiredancer,
			cluster:    constants.ClusterNameTestnet,
			regexType:  "ReleaseTitleRegex",
			regex:      "^(.*)dancer Testnet v([0-9]+\\.[0-9]+\\.[0-9]+)(?:\\b.*)?$",
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

func TestFiredancerCompatibilityKey(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	tests := []struct {
		name string
		in   string
		want int64
	}{
		{
			name: "repo tag with feature-set patch",
			in:   "v0.910.40000",
			want: 40000,
		},
		{
			name: "newer repo tag with feature-set patch",
			in:   "v0.1001.40101",
			want: 40101,
		},
		{
			name: "SFDP beta-shaped compatibility version",
			in:   "0.101.0-beta.40101",
			want: 40101,
		},
		{
			name: "RPC beta-shaped compatibility version",
			in:   "0.902.0-beta.40002",
			want: 40002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := firedancerCompatibilityKey(mustVersion(tt.in))
			if err != nil {
				t.Fatalf("firedancerCompatibilityKey() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("firedancerCompatibilityKey(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestFiredancerCompatibilityVersionKey(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	tests := []struct {
		name              string
		in                string
		wantTrain         int64
		wantCompatibility int64
	}{
		{
			name:              "repo tag uses minor as release train",
			in:                "v0.1005.40100",
			wantTrain:         1005,
			wantCompatibility: 40100,
		},
		{
			name:              "current SFDP rc-shaped min uses minor as release train",
			in:                "0.1004.0-rc.40101",
			wantTrain:         1004,
			wantCompatibility: 40101,
		},
		{
			name:              "legacy SFDP beta-shaped min maps to repo train",
			in:                "0.101.0-beta.40101",
			wantTrain:         1001,
			wantCompatibility: 40101,
		},
		{
			name:              "older RPC beta-shaped version keeps repo train",
			in:                "0.902.0-beta.40002",
			wantTrain:         902,
			wantCompatibility: 40002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := firedancerCompatibilityVersionKey(mustVersion(tt.in))
			if err != nil {
				t.Fatalf("firedancerCompatibilityVersionKey() error = %v", err)
			}
			if got.train != tt.wantTrain || got.compatibilityKey != tt.wantCompatibility {
				t.Errorf("firedancerCompatibilityVersionKey(%q) = {%d %d}, want {%d %d}",
					tt.in,
					got.train,
					got.compatibilityKey,
					tt.wantTrain,
					tt.wantCompatibility,
				)
			}
		})
	}
}

func TestFiredancerCompatibilityVersionKeyCompare(t *testing.T) {
	mustKey := func(s string) firedancerCompatibilityKeyTuple {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		key, err := firedancerCompatibilityVersionKey(v)
		if err != nil {
			t.Fatalf("firedancerCompatibilityVersionKey(%q) error = %v", s, err)
		}
		return key
	}

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "newer mainnet train satisfies lower SFDP train despite lower compatibility key",
			a:    "v0.1005.40100",
			b:    "0.1004.0-rc.40101",
			want: 1,
		},
		{
			name: "same train compares by compatibility key",
			a:    "v0.1004.40101",
			b:    "0.1004.0-rc.40101",
			want: 0,
		},
		{
			name: "legacy SFDP beta min maps to matching repo train",
			a:    "v0.1001.40101",
			b:    "0.101.0-beta.40101",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustKey(tt.a).Compare(mustKey(tt.b))
			if got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
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
			if got.String() != mustVersion(tt.want).String() {
				t.Errorf("NormalizeToTagVersion(%q) = %q, want %q", tt.input, got.String(), tt.want)
			}
		})
	}
}

func TestResolveFiredancerSFDPCompliantVersion(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	clientWithTags := func(tags ...string) *Client {
		tagInfos := make([]tagVersionInfo, 0, len(tags))
		tagVersions := make([]*goversion.Version, 0, len(tags))
		for _, tag := range tags {
			v := mustVersion(tag)
			tagInfos = append(tagInfos, tagVersionInfo{
				TagName: tag,
				Version: v,
			})
			tagVersions = append(tagVersions, v)
		}
		return &Client{
			clientName:        constants.ClientNameFiredancer,
			cachedTagInfos:    tagInfos,
			cachedTagVersions: tagVersions,
			logger:            log.WithPrefix("test"),
		}
	}

	tests := []struct {
		name      string
		tags      []string
		target    string
		min       string
		hasMin    bool
		max       string
		hasMax    bool
		want      string
		wantError bool
	}{
		{
			name:   "maps SFDP beta min to matching repo tag",
			tags:   []string{"v0.910.40000", "v0.1001.40101"},
			target: "v0.910.40000",
			min:    "0.101.0-beta.40101",
			hasMin: true,
			want:   "v0.1001.40101",
		},
		{
			name:   "keeps target when compatibility key satisfies min",
			tags:   []string{"v0.910.40000", "v0.1001.40101"},
			target: "v0.1001.40101",
			min:    "0.101.0-beta.40101",
			hasMin: true,
			want:   "v0.1001.40101",
		},
		{
			name:   "keeps newer mainnet train when SFDP min has higher compatibility key on older train",
			tags:   []string{"v0.1004.40101", "v0.1005.40100"},
			target: "v0.1005.40100",
			min:    "0.1004.0-rc.40101",
			hasMin: true,
			want:   "v0.1005.40100",
		},
		{
			name:      "errors when no cached repo tag satisfies min",
			tags:      []string{"v0.910.40000"},
			target:    "v0.910.40000",
			min:       "0.101.0-beta.40101",
			hasMin:    true,
			wantError: true,
		},
		{
			name:   "selects highest compatible tag for max bound",
			tags:   []string{"v0.909.39999", "v0.910.40000", "v0.1001.40101"},
			target: "v0.1001.40101",
			max:    "0.910.40000",
			hasMax: true,
			want:   "v0.910.40000",
		},
		{
			name:   "uses latest tag when duplicate compatibility keys exist",
			tags:   []string{"v0.1000.40101", "v0.1001.40101"},
			target: "v0.910.40000",
			min:    "0.101.0-beta.40101",
			hasMin: true,
			want:   "v0.1001.40101",
		},
		{
			name:   "keeps native firedancer target when SFDP has legacy frankendancer min",
			tags:   []string{"v0.1001.40101", "v1.0.0"},
			target: "v1.0.0",
			min:    "0.101.0-beta.40101",
			hasMin: true,
			want:   "v1.0.0",
		},
		{
			name:      "errors for native firedancer target with explicit max bound",
			tags:      []string{"v0.1001.40101", "v1.0.0"},
			target:    "v1.0.0",
			max:       "0.1001.40101",
			hasMax:    true,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var minVersion *goversion.Version
			if tt.hasMin {
				minVersion = mustVersion(tt.min)
			}
			var maxVersion *goversion.Version
			if tt.hasMax {
				maxVersion = mustVersion(tt.max)
			}

			got, err := clientWithTags(tt.tags...).ResolveFiredancerSFDPCompliantVersion(
				mustVersion(tt.target),
				minVersion,
				tt.hasMin,
				maxVersion,
				tt.hasMax,
			)
			if (err != nil) != tt.wantError {
				t.Fatalf("ResolveFiredancerSFDPCompliantVersion() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError {
				return
			}
			if got.Original() != tt.want {
				t.Errorf("ResolveFiredancerSFDPCompliantVersion() = %q, want %q", got.Original(), tt.want)
			}
		})
	}
}

func TestResolveFiredancerSFDPCompliantVersionDoesNotSelectTestnetOnlyTagForMainnet(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	client := &Client{
		clientName: constants.ClientNameFiredancer,
		cluster:    constants.ClusterNameMainnetBeta,
		cachedTagInfos: []tagVersionInfo{
			{TagName: "v0.910.40000", Version: mustVersion("v0.910.40000")},
			{TagName: "v0.1001.40101", Version: mustVersion("v0.1001.40101"), TestnetOnly: true},
		},
		logger: log.WithPrefix("test"),
	}

	_, err := client.ResolveFiredancerSFDPCompliantVersion(
		mustVersion("v0.910.40000"),
		mustVersion("0.101.0-beta.40101"),
		true,
		nil,
		false,
	)
	if err == nil {
		t.Fatal("ResolveFiredancerSFDPCompliantVersion() should not select a testnet-only tag for mainnet")
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

func TestTagNameForVersion_JitoSolana(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	client := &Client{
		clientName: constants.ClientNameJitoSolana,
		cachedTagInfos: []tagVersionInfo{
			{TagName: "v4.1.0-beta.1-jito", Version: mustVersion("v4.1.0-beta.1")},
			{TagName: "v3.0.6-jito.1", Version: mustVersion("v3.0.6")},
		},
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "pre-release agave version maps to full jito tag",
			input: "4.1.0-beta.1",
			want:  "v4.1.0-beta.1-jito",
		},
		{
			name:  "stable agave version maps to jito patch suffix tag",
			input: "3.0.6",
			want:  "v3.0.6-jito.1",
		},
		{
			name:  "unknown version falls back unchanged",
			input: "4.1.0-beta.2",
			want:  "4.1.0-beta.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.TagNameForVersion(mustVersion(tt.input)); got != tt.want {
				t.Errorf("TagNameForVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasTaggedVersion_JitoSolanaCachesMatchingTag(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	tests := []struct {
		name    string
		tags    string
		target  string
		wantHas bool
		wantTag string
	}{
		{
			name:    "caches exact jito pre-release tag",
			tags:    `[{"name":"v4.1.0-beta.1-jito"}]`,
			target:  "4.1.0-beta.1",
			wantHas: true,
			wantTag: "v4.1.0-beta.1-jito",
		},
		{
			name:    "does not match different jito pre-release by core only",
			tags:    `[{"name":"v4.1.0-beta.2-jito"}]`,
			target:  "4.1.0-beta.1",
			wantHas: false,
			wantTag: "4.1.0-beta.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path != "/repos/jito-foundation/jito-solana/tags" {
						return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(tt.tags)),
						Request:    r,
					}, nil
				}),
			}

			ghClient := gogithub.NewClient(httpClient)
			baseURL, err := url.Parse("https://api.github.test/")
			if err != nil {
				t.Fatalf("failed to parse test GitHub API URL: %v", err)
			}
			ghClient.BaseURL = baseURL

			client := &Client{
				clientName: constants.ClientNameJitoSolana,
				repoOwner:  "jito-foundation",
				repoName:   "jito-solana",
				client:     ghClient,
				logger:     log.WithPrefix("test"),
			}

			has, err := client.HasTaggedVersion(mustVersion(tt.target))
			if err != nil {
				t.Fatalf("HasTaggedVersion() error = %v", err)
			}
			if has != tt.wantHas {
				t.Fatalf("HasTaggedVersion() = %v, want %v", has, tt.wantHas)
			}
			if got := client.TagNameForVersion(mustVersion(tt.target)); got != tt.wantTag {
				t.Errorf("TagNameForVersion() = %q, want %q", got, tt.wantTag)
			}
		})
	}
}

func TestHasTaggedVersion_AgavePrereleaseRequiresExactMatch(t *testing.T) {
	mustVersion := func(s string) *goversion.Version {
		v, err := goversion.NewVersion(s)
		if err != nil {
			t.Fatalf("failed to parse version %q: %v", s, err)
		}
		return v
	}

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/repos/anza-xyz/agave/tags" {
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`[{"name":"v4.2.0-beta.2"},{"name":"v4.1.2"}]`)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client := &Client{
		clientName: constants.ClientNameAgave,
		repoOwner:  "anza-xyz",
		repoName:   "agave",
		client:     ghClient,
		logger:     log.WithPrefix("test"),
	}

	has, err := client.HasTaggedVersion(mustVersion("v4.2.0-beta.1"))
	if err != nil {
		t.Fatalf("HasTaggedVersion() error = %v", err)
	}
	if has {
		t.Fatal("HasTaggedVersion() matched prerelease by core version; want exact prerelease match only")
	}

	has, err = client.HasTaggedVersion(mustVersion("v4.1.2"))
	if err != nil {
		t.Fatalf("HasTaggedVersion() error = %v", err)
	}
	if !has {
		t.Fatal("HasTaggedVersion() = false for stable core match, want true")
	}
}

func TestGetLatestClientVersion_JitoSolanaIncludesTestnetPrereleases(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body string
			switch r.URL.Path {
			case "/repos/jito-foundation/jito-solana/releases":
				body = `[
					{"name":"Mainnet - v4.0.2-jito","tag_name":"v4.0.2-jito","prerelease":false},
					{"name":"Testnet - v4.1.0-beta.3-jito","tag_name":"v4.1.0-beta.3-jito","prerelease":true},
					{"name":"Testnet - v4.0.0-jito","tag_name":"v4.0.0-jito","prerelease":false}
				]`
			case "/repos/anza-xyz/agave/releases":
				body = `[
					{"name":"Release v4.0.2","tag_name":"v4.0.2","body":"This is a stable release suitable for use on Mainnet Beta.","prerelease":false},
					{"name":"Release v4.1.0-beta.3","tag_name":"v4.1.0-beta.3","body":"This is a Testnet release.","prerelease":true},
					{"name":"Release v4.0.0","tag_name":"v4.0.0","body":"This is a stable release suitable for Testnet and Devnet.","prerelease":false}
				]`
			default:
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client := &Client{
		clientName: constants.ClientNameJitoSolana,
		cluster:    constants.ClusterNameTestnet,
		repoOwner:  "jito-foundation",
		repoName:   "jito-solana",
		repoURL:    clientRepoConfigs[constants.ClientNameJitoSolana].URL,
		client:     ghClient,
		logger:     log.WithPrefix("test"),
	}

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	want, err := goversion.NewVersion("v4.1.0-beta.3")
	if err != nil {
		t.Fatalf("failed to parse wanted version: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), want.Original())
	}
	if gotTag := client.TagNameForVersion(got); gotTag != "v4.1.0-beta.3-jito" {
		t.Errorf("TagNameForVersion() = %q, want %q", gotTag, "v4.1.0-beta.3-jito")
	}
}

func TestGetLatestClientVersion_JitoSolanaUsesTestnetTitleWhenAgaveNotesOmitCluster(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body string
			switch r.URL.Path {
			case "/repos/jito-foundation/jito-solana/releases":
				body = `[
					{"name":"Testnet - v4.2.0-beta.1-jito","tag_name":"v4.2.0-beta.1-jito","prerelease":true},
					{"name":"Mainnet - v4.1.2-jito","tag_name":"v4.1.2-jito","prerelease":false},
					{"name":"Testnet - Release v4.2.0-beta.0-jito","tag_name":"v4.2.0-beta.0-jito","prerelease":true}
				]`
			case "/repos/anza-xyz/agave/releases":
				body = `[
					{"name":"Release v4.2.0-beta.1","tag_name":"v4.2.0-beta.1","body":"## What's Changed\n* v4.2: runtime backport\n* v4.2: gossip backport","prerelease":true},
					{"name":"Release v4.1.2","tag_name":"v4.1.2","body":"This is a stable Mainnet release.","prerelease":false},
					{"name":"Release v4.2.0-beta.0","tag_name":"v4.2.0-beta.0","body":"This is a testnet release.","prerelease":true}
				]`
			default:
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client := &Client{
		clientName: constants.ClientNameJitoSolana,
		cluster:    constants.ClusterNameTestnet,
		repoOwner:  "jito-foundation",
		repoName:   "jito-solana",
		repoURL:    clientRepoConfigs[constants.ClientNameJitoSolana].URL,
		client:     ghClient,
		logger:     log.WithPrefix("test"),
	}

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	want, err := goversion.NewVersion("v4.2.0-beta.1")
	if err != nil {
		t.Fatalf("failed to parse wanted version: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), want.Original())
	}
	if gotTag := client.TagNameForVersion(got); gotTag != "v4.2.0-beta.1-jito" {
		t.Errorf("TagNameForVersion() = %q, want %q", gotTag, "v4.2.0-beta.1-jito")
	}
}

func TestGetLatestClientVersion_JitoSolanaPrefersMainnetTitleOverAgaveDerivedCandidate(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body string
			switch r.URL.Path {
			case "/repos/jito-foundation/jito-solana/releases":
				body = `[
					{"name":"Mainnet - v4.0.2-jito","tag_name":"v4.0.2-jito","prerelease":false},
					{"name":"Testnet - v4.1.0-rc.1-jito","tag_name":"v4.1.0-rc.1-jito","prerelease":true}
				]`
			case "/repos/anza-xyz/agave/releases":
				body = `[
					{"name":"Release v4.0.2","tag_name":"v4.0.2","body":"This is a stable release suitable for use on Mainnet Beta.","prerelease":false},
					{"name":"Release v4.1.0-rc.1","tag_name":"v4.1.0-rc.1","body":"Mainnet Upgrade Candidate. It is also recommended for Testnet and Devnet.","prerelease":true}
				]`
			default:
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client := &Client{
		clientName: constants.ClientNameJitoSolana,
		cluster:    constants.ClusterNameMainnetBeta,
		repoOwner:  "jito-foundation",
		repoName:   "jito-solana",
		repoURL:    clientRepoConfigs[constants.ClientNameJitoSolana].URL,
		client:     ghClient,
		logger:     log.WithPrefix("test"),
	}

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	want, err := goversion.NewVersion("v4.0.2")
	if err != nil {
		t.Fatalf("failed to parse wanted version: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), want.Original())
	}
	if gotTag := client.TagNameForVersion(got); gotTag != "v4.0.2-jito" {
		t.Errorf("TagNameForVersion() = %q, want %q", gotTag, "v4.0.2-jito")
	}
}

func TestGetLatestClientVersion_JitoSolanaFallsBackToAgaveDerivedCandidateWhenTitleMissing(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body string
			switch r.URL.Path {
			case "/repos/jito-foundation/jito-solana/releases":
				body = `[
					{"name":"Release v4.2.0-beta.1-jito","tag_name":"v4.2.0-beta.1-jito","prerelease":true},
					{"name":"Mainnet - v4.1.2-jito","tag_name":"v4.1.2-jito","prerelease":false}
				]`
			case "/repos/anza-xyz/agave/releases":
				body = `[
					{"name":"Release v4.2.0-beta.1","tag_name":"v4.2.0-beta.1","body":"This is a testnet release.","prerelease":true},
					{"name":"Release v4.1.2","tag_name":"v4.1.2","body":"This is a stable Mainnet release.","prerelease":false}
				]`
			default:
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client := &Client{
		clientName: constants.ClientNameJitoSolana,
		cluster:    constants.ClusterNameTestnet,
		repoOwner:  "jito-foundation",
		repoName:   "jito-solana",
		repoURL:    clientRepoConfigs[constants.ClientNameJitoSolana].URL,
		client:     ghClient,
		logger:     log.WithPrefix("test"),
	}

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	want, err := goversion.NewVersion("v4.2.0-beta.1")
	if err != nil {
		t.Fatalf("failed to parse wanted version: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), want.Original())
	}
	if gotTag := client.TagNameForVersion(got); gotTag != "v4.2.0-beta.1-jito" {
		t.Errorf("TagNameForVersion() = %q, want %q", gotTag, "v4.2.0-beta.1-jito")
	}
}

func TestGetLatestClientVersion_AgaveIgnoresTestnetPrereleaseWhenNotesOmitCluster(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/repos/anza-xyz/agave/releases" {
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}
			body := `[
				{"name":"Release v4.3.0-alpha.1","tag_name":"v4.3.0-alpha.1","body":"## What's Changed\n* master branch changes","prerelease":true},
				{"name":"Release v4.2.0-beta.1","tag_name":"v4.2.0-beta.1","body":"## What's Changed\n* v4.2: runtime backport\n* v4.2: gossip backport","prerelease":true},
				{"name":"Release v4.1.2","tag_name":"v4.1.2","body":"This is a stable Mainnet release.","prerelease":false},
				{"name":"Release v4.2.0-beta.0","tag_name":"v4.2.0-beta.0","body":"This is a testnet release.","prerelease":true}
			]`

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client, err := NewClient(Options{
		Client:  constants.ClientNameAgave,
		Cluster: constants.ClusterNameTestnet,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.client = ghClient

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	want, err := goversion.NewVersion("v4.2.0-beta.0")
	if err != nil {
		t.Fatalf("failed to parse wanted version: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), want.Original())
	}
}

func TestGetLatestClientVersion_AgaveUsesExplicitTestnetPrereleaseNotes(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/repos/anza-xyz/agave/releases" {
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}
			body := `[
				{"name":"Release v4.3.0-alpha.2","tag_name":"v4.3.0-alpha.2","body":"## What's Changed\n* master branch changes","prerelease":true},
				{"name":"Release v4.2.0-beta.2","tag_name":"v4.2.0-beta.2","body":"This is a testnet release.","prerelease":true},
				{"name":"Release v4.1.2","tag_name":"v4.1.2","body":"This is a stable Mainnet release.","prerelease":false}
			]`

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	client, err := NewClient(Options{
		Client:  constants.ClientNameAgave,
		Cluster: constants.ClusterNameTestnet,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.client = ghClient

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	want, err := goversion.NewVersion("v4.2.0-beta.2")
	if err != nil {
		t.Fatalf("failed to parse wanted version: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), want.Original())
	}
}

func TestGetLatestClientVersion_FiredancerIncludesMainnetSuitablePrerelease(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/repos/firedancer-io/firedancer/releases" {
				return nil, fmt.Errorf("unexpected request path %q", r.URL.Path)
			}

			body := `[
				{"name":"Frankendancer Mainnet v0.1005.40100","tag_name":"v0.1005.40100","body":"This is a mainnet ready release.","prerelease":false},
				{"name":"Frankendancer Mainnet v0.909.40001","tag_name":"v0.909.40001","body":"This is a mainnet release.","prerelease":false},
				{"name":"Frankendancer Testnet v0.1002.40103","tag_name":"v0.1002.40103","body":"This is a Testnet release.","prerelease":true},
				{"name":"Frankendancer Testnet v0.1004.40101","tag_name":"v0.1004.40101","body":"This is a Testnet release. It may also be used on mainnet with a small amount of stake in accordance with Anza's guidelines for v4.1.0-rc.1.","prerelease":true}
			]`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	client, err := NewClient(Options{
		Cluster: constants.ClusterNameMainnetBeta,
		Client:  constants.ClientNameFiredancer,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ghClient := gogithub.NewClient(httpClient)
	baseURL, err := url.Parse("https://api.github.test/")
	if err != nil {
		t.Fatalf("failed to parse test GitHub API URL: %v", err)
	}
	ghClient.BaseURL = baseURL
	client.client = ghClient

	got, err := client.GetLatestClientVersion()
	if err != nil {
		t.Fatalf("GetLatestClientVersion() error = %v", err)
	}
	if got.Original() != "v0.1005.40100" {
		t.Fatalf("GetLatestClientVersion() = %q, want %q", got.Original(), "v0.1005.40100")
	}
	if gotTag := client.TagNameForVersion(got); gotTag != "v0.1005.40100" {
		t.Errorf("TagNameForVersion() = %q, want %q", gotTag, "v0.1005.40100")
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
			name:            "Testnet release prefix",
			cluster:         constants.ClusterNameTestnet,
			releaseTitle:    "Testnet - Release v4.2.0-beta.0-jito",
			shouldMatch:     true,
			expectedVersion: "4.2.0-beta.0",
		},
		{
			name:            "Mainnet extra spacing",
			cluster:         constants.ClusterNameMainnetBeta,
			releaseTitle:    "Mainnet -  v4.0.0-jito",
			shouldMatch:     true,
			expectedVersion: "4.0.0",
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
