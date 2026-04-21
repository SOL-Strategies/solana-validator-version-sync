package constants

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// ClientNameAgave is the name of the Agave client
	ClientNameAgave = "agave"
	// ClientNameJitoSolana is the name of the Jito Solana client
	ClientNameJitoSolana = "jito-solana"
	// ClientNameRakurai is the canonical name of the Rakurai validator client
	ClientNameRakurai = "rakurai-validator"
	// ClientNameFiredancer is the name of the Firedancer client
	ClientNameFiredancer = "firedancer"
	// ClusterNameMainnetBeta is the name of the Mainnet Beta cluster
	ClusterNameMainnetBeta = "mainnet-beta"
	// ClusterNameTestnet is the name of the Testnet cluster
	ClusterNameTestnet = "testnet"

	// clientNameRakuraiAlias is the legacy Rakurai client name kept for backwards compatibility
	clientNameRakuraiAlias = "rakurai"
)

// ValidClientNames is a list of valid canonical client names
var ValidClientNames = []string{ClientNameAgave, ClientNameJitoSolana, ClientNameRakurai, ClientNameFiredancer}

// ValidClusterNames is a list of valid cluster names
var ValidClusterNames = []string{ClusterNameMainnetBeta, ClusterNameTestnet}

// NormalizeClientName maps legacy client names to their canonical form.
func NormalizeClientName(clientName string) string {
	switch clientName {
	case clientNameRakuraiAlias:
		return ClientNameRakurai
	default:
		return clientName
	}
}

// ValidateClientName validates a client name
func ValidateClientName(clientName string) (err error) {
	if !slices.Contains(ValidClientNames, NormalizeClientName(clientName)) {
		return fmt.Errorf("invalid client name: %s - must be one of %s", clientName, strings.Join(ValidClientNames, ", "))
	}
	return nil
}

// ValidateClusterName validates a cluster name
func ValidateClusterName(clusterName string) (err error) {
	if !slices.Contains(ValidClusterNames, clusterName) {
		return fmt.Errorf("invalid cluster name: %s - must be one of %s", clusterName, strings.Join(ValidClusterNames, ", "))
	}
	return nil
}
