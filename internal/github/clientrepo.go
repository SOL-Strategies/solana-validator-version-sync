package github

import "github.com/sol-strategies/solana-validator-version-sync/internal/constants"

// ClientRepoConfig represents the configuration for a client source repository
type ClientRepoConfig struct {
	URL                 string
	ReleaseNotesRegexes map[string]string
	ReleaseTitleRegexes map[string]string
	TagRegexes          map[string]string
}

var clientRepoConfigs = map[string]ClientRepoConfig{
	constants.ClientNameAgave: {
		URL: "https://github.com/anza-xyz/agave",
		ReleaseNotesRegexes: map[string]string{
			constants.ClusterNameMainnetBeta: ".*(This is a stable release suitable for use on Mainnet Beta|This (?:is )?a stable Mainnet release).*",
			constants.ClusterNameTestnet:     "(?is).*(This is a testnet release|recommended for testnet).*",
		},
	},
	constants.ClientNameJitoSolana: {
		URL: "https://github.com/jito-foundation/jito-solana",
		ReleaseTitleRegexes: map[string]string{
			constants.ClusterNameMainnetBeta: "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?)-jito(?:\\.[0-9]+)?$",
			constants.ClusterNameTestnet:     "^Testnet - v([0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?)-jito(?:\\.[0-9]+)?$",
		},
	},
	constants.ClientNameRakurai: {
		URL: "https://github.com/rakurai-io/rakurai-validator",
		TagRegexes: map[string]string{
			// Rakurai publishes release tags from the rakurai-validator repo.
			// We intentionally ignore ".b" variants for now until Rakurai documents
			// their semantics more clearly.
			constants.ClusterNameMainnetBeta: "^release/(v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?-rakurai\\.[0-9]+)$",
			constants.ClusterNameTestnet:     "^release/(v[0-9]+\\.[0-9]+\\.[0-9]+(?:-[a-zA-Z][a-zA-Z0-9.]*)?-rakurai\\.[0-9]+)_testnet$",
		},
	},
	constants.ClientNameFiredancer: {
		URL: "https://github.com/firedancer-io/firedancer",
		ReleaseTitleRegexes: map[string]string{
			// One day this will change from Frankendancer to Firedancer so we match on dancer suffix
			constants.ClusterNameMainnetBeta: "^(.*)dancer Mainnet v([0-9]+\\.[0-9]+\\.[0-9]+)$",
			// One day this will change from Frankendancer to Firedancer so we match on dancer suffix
			constants.ClusterNameTestnet: "^(.*)dancer Testnet v([0-9]+\\.[0-9]+\\.[0-9]+)$",
		},
	},
}
