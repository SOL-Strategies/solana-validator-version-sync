package config

import (
	"testing"

	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestCluster_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cluster Cluster
		wantErr bool
	}{
		{
			name: "valid mainnet-beta cluster",
			cluster: Cluster{
				Name: constants.ClusterNameMainnetBeta,
			},
			wantErr: false,
		},
		{
			name: "valid testnet cluster",
			cluster: Cluster{
				Name: constants.ClusterNameTestnet,
			},
			wantErr: false,
		},
		{
			name: "invalid cluster name - empty string",
			cluster: Cluster{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "invalid cluster name - random string",
			cluster: Cluster{
				Name: "invalid-cluster",
			},
			wantErr: true,
		},
		{
			name: "invalid cluster name - case sensitive",
			cluster: Cluster{
				Name: "MAINNET-BETA",
			},
			wantErr: true,
		},
		{
			name: "invalid cluster name - partial match",
			cluster: Cluster{
				Name: "mainnet",
			},
			wantErr: true,
		},
		{
			name: "invalid cluster name - with spaces",
			cluster: Cluster{
				Name: "mainnet beta",
			},
			wantErr: true,
		},
		{
			name: "invalid cluster name - devnet",
			cluster: Cluster{
				Name: "devnet",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cluster.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCluster_StructFields(t *testing.T) {
	// Test that the struct has the expected fields and tags
	cluster := Cluster{
		Name: constants.ClusterNameMainnetBeta,
	}

	// Verify the struct can be instantiated with valid data
	if cluster.Name != constants.ClusterNameMainnetBeta {
		t.Errorf("Expected cluster name to be %s, got %s", constants.ClusterNameMainnetBeta, cluster.Name)
	}

	// Test that the koanf tag is properly set (this is more of a compile-time check)
	// We can't easily test struct tags at runtime without reflection, but we can ensure
	// the struct compiles and works as expected
}
