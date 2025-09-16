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
	// ClientNameFiredancer is the name of the Firedancer client
	ClientNameFiredancer = "firedancer"
	// ClusterNameMainnetBeta is the name of the Mainnet Beta cluster
	ClusterNameMainnetBeta = "mainnet-beta"
	// ClusterNameTestnet is the name of the Testnet cluster
	ClusterNameTestnet = "testnet"
)

// ValidClientNames is a list of valid client names
var ValidClientNames = []string{ClientNameAgave, ClientNameJitoSolana, ClientNameFiredancer}

// ValidClusterNames is a list of valid cluster names
var ValidClusterNames = []string{ClusterNameMainnetBeta, ClusterNameTestnet}

// ValidateClientName validates a client name
func ValidateClientName(clientName string) (err error) {
	if !slices.Contains(ValidClientNames, clientName) {
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
