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
	// ClientNameFireBAM is the name of the BAM-enabled Firedancer client
	ClientNameFireBAM = "firebam"
	// ReleaseTrackFrankendancer selects hybrid Frankendancer releases
	ReleaseTrackFrankendancer = "frankendancer"
	// ReleaseTrackFiredancer selects native Firedancer releases
	ReleaseTrackFiredancer = "firedancer"
	// ClusterNameMainnetBeta is the name of the Mainnet Beta cluster
	ClusterNameMainnetBeta = "mainnet-beta"
	// ClusterNameTestnet is the name of the Testnet cluster
	ClusterNameTestnet = "testnet"

	// clientNameRakuraiAlias is the legacy Rakurai client name kept for backwards compatibility
	clientNameRakuraiAlias = "rakurai"
)

// ValidClientNames is a list of valid canonical client names
var ValidClientNames = []string{ClientNameAgave, ClientNameJitoSolana, ClientNameRakurai, ClientNameFiredancer, ClientNameFireBAM}

// ValidFireBAMReleaseTracks is a list of release tracks supported by FireBAM.
var ValidFireBAMReleaseTracks = []string{ReleaseTrackFrankendancer, ReleaseTrackFiredancer}

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

// ValidateReleaseTrack validates the FireBAM-only release track setting.
func ValidateReleaseTrack(clientName string, releaseTrack string) error {
	clientName = NormalizeClientName(clientName)
	if clientName == ClientNameFireBAM {
		if !slices.Contains(ValidFireBAMReleaseTracks, releaseTrack) {
			return fmt.Errorf("validator.release_track is required for firebam and must be one of %s", strings.Join(ValidFireBAMReleaseTracks, ", "))
		}
		return nil
	}
	if releaseTrack != "" {
		return fmt.Errorf("validator.release_track is only supported when validator.client is firebam")
	}
	return nil
}

// IsFiredancerFamily reports whether a client uses Firedancer-family versioning and SFDP requirements.
func IsFiredancerFamily(clientName string) bool {
	clientName = NormalizeClientName(clientName)
	return clientName == ClientNameFiredancer || clientName == ClientNameFireBAM
}

// ValidateClusterName validates a cluster name
func ValidateClusterName(clusterName string) (err error) {
	if !slices.Contains(ValidClusterNames, clusterName) {
		return fmt.Errorf("invalid cluster name: %s - must be one of %s", clusterName, strings.Join(ValidClusterNames, ", "))
	}
	return nil
}
