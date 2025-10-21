package github

import "github.com/sol-strategies/solana-validator-version-sync/internal/constants"

// ClientRepoConfig represents the configuration for a client source repository
type ClientRepoConfig struct {
	URL                 string
	ReleaseNotesRegexes map[string]string
	ReleaseTitleRegexes map[string]string
}

var clientRepoConfigs = map[string]ClientRepoConfig{
	constants.ClientNameAgave: {
		URL: "https://github.com/anza-xyz/agave",
		ReleaseNotesRegexes: map[string]string{
			constants.ClusterNameMainnetBeta: ".*This is a stable release suitable for use on Mainnet Beta.*",
			constants.ClusterNameTestnet:     ".*This is a Testnet release.*",
		},
	},
	constants.ClientNameJitoSolana: {
		URL: "https://github.com/jito-foundation/jito-solana",
		ReleaseTitleRegexes: map[string]string{
			constants.ClusterNameMainnetBeta: "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito(?:\\.([0-9]+))?$",
			constants.ClusterNameTestnet:     "^Testnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-jito(?:\\.([0-9]+))?$",
		},
	},
	constants.ClientNameBAM: {
		URL: "https://github.com/jito-labs/bam-client",
		ReleaseTitleRegexes: map[string]string{
			constants.ClusterNameMainnetBeta: "^Mainnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-bam$",
			constants.ClusterNameTestnet:     "^Testnet - v([0-9]+\\.[0-9]+\\.[0-9]+)-bam$",
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
