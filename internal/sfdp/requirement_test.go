package sfdp

import (
	"testing"

	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestRequirements_StructFields(t *testing.T) {
	req := Requirements{
		Epoch:                      500,
		Cluster:                    "mainnet-beta",
		AgaveMinVersion:            "1.18.0",
		AgaveMaxVersion:            "1.18.5",
		FiredancerMinVersion:       "0.1.0",
		FiredancerMaxVersion:       "0.1.2",
		InheritedFromPreviousEpoch: false,
		Client:                     constants.ClientNameAgave,
		ConstraintsString:          ">= 1.18.0, <= 1.18.5",
		HasMaxVersion:              true,
		HasMinVersion:              true,
	}

	if req.Epoch != 500 {
		t.Errorf("Expected Epoch to be 500, got %d", req.Epoch)
	}
	if req.Cluster != "mainnet-beta" {
		t.Errorf("Expected Cluster to be mainnet-beta, got %s", req.Cluster)
	}
	if req.AgaveMinVersion != "1.18.0" {
		t.Errorf("Expected AgaveMinVersion to be 1.18.0, got %s", req.AgaveMinVersion)
	}
	if req.AgaveMaxVersion != "1.18.5" {
		t.Errorf("Expected AgaveMaxVersion to be 1.18.5, got %s", req.AgaveMaxVersion)
	}
	if req.FiredancerMinVersion != "0.1.0" {
		t.Errorf("Expected FiredancerMinVersion to be 0.1.0, got %s", req.FiredancerMinVersion)
	}
	if req.FiredancerMaxVersion != "0.1.2" {
		t.Errorf("Expected FiredancerMaxVersion to be 0.1.2, got %s", req.FiredancerMaxVersion)
	}
	if req.InheritedFromPreviousEpoch != false {
		t.Errorf("Expected InheritedFromPreviousEpoch to be false, got %v", req.InheritedFromPreviousEpoch)
	}
	if req.Client != constants.ClientNameAgave {
		t.Errorf("Expected Client to be %s, got %s", constants.ClientNameAgave, req.Client)
	}
	if req.ConstraintsString != ">= 1.18.0, <= 1.18.5" {
		t.Errorf("Expected ConstraintsString to be '>= 1.18.0, <= 1.18.5', got %s", req.ConstraintsString)
	}
	if req.HasMaxVersion != true {
		t.Errorf("Expected HasMaxVersion to be true, got %v", req.HasMaxVersion)
	}
	if req.HasMinVersion != true {
		t.Errorf("Expected HasMinVersion to be true, got %v", req.HasMinVersion)
	}
}

func TestRequirements_SetClient(t *testing.T) {
	tests := []struct {
		name                 string
		client               string
		agaveMinVersion      string
		agaveMaxVersion      string
		firedancerMinVersion string
		firedancerMaxVersion string
		wantErr              bool
		expectedClient       string
		expectedMinVersion   string
		expectedMaxVersion   string
		expectedHasMin       bool
		expectedHasMax       bool
	}{
		{
			name:                 "agave client with min and max versions",
			client:               constants.ClientNameAgave,
			agaveMinVersion:      "1.18.0",
			agaveMaxVersion:      "1.18.5",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              false,
			expectedClient:       constants.ClientNameAgave,
			expectedMinVersion:   "1.18.0",
			expectedMaxVersion:   "1.18.5",
			expectedHasMin:       true,
			expectedHasMax:       true,
		},
		{
			name:                 "jito-solana client (should map to agave)",
			client:               constants.ClientNameJitoSolana,
			agaveMinVersion:      "1.18.0",
			agaveMaxVersion:      "1.18.5",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              false,
			expectedClient:       constants.ClientNameAgave,
			expectedMinVersion:   "1.18.0",
			expectedMaxVersion:   "1.18.5",
			expectedHasMin:       true,
			expectedHasMax:       true,
		},
		{
			name:                 "firedancer client with min and max versions",
			client:               constants.ClientNameFiredancer,
			agaveMinVersion:      "1.18.0",
			agaveMaxVersion:      "1.18.5",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              false,
			expectedClient:       constants.ClientNameFiredancer,
			expectedMinVersion:   "0.1.0",
			expectedMaxVersion:   "0.1.2",
			expectedHasMin:       true,
			expectedHasMax:       true,
		},
		{
			name:                 "agave client with only min version",
			client:               constants.ClientNameAgave,
			agaveMinVersion:      "1.18.0",
			agaveMaxVersion:      "",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              false,
			expectedClient:       constants.ClientNameAgave,
			expectedMinVersion:   "1.18.0",
			expectedMaxVersion:   "",
			expectedHasMin:       true,
			expectedHasMax:       false,
		},
		{
			name:                 "agave client with only max version",
			client:               constants.ClientNameAgave,
			agaveMinVersion:      "",
			agaveMaxVersion:      "1.18.5",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              false,
			expectedClient:       constants.ClientNameAgave,
			expectedMinVersion:   "",
			expectedMaxVersion:   "1.18.5",
			expectedHasMin:       false,
			expectedHasMax:       true,
		},
		{
			name:                 "agave client with no versions",
			client:               constants.ClientNameAgave,
			agaveMinVersion:      "",
			agaveMaxVersion:      "",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              true, // This will fail due to empty constraint string
			expectedClient:       constants.ClientNameAgave,
			expectedMinVersion:   "",
			expectedMaxVersion:   "",
			expectedHasMin:       false,
			expectedHasMax:       false,
		},
		{
			name:                 "invalid client",
			client:               "invalid-client",
			agaveMinVersion:      "1.18.0",
			agaveMaxVersion:      "1.18.5",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              true,
		},
		{
			name:                 "invalid min version format",
			client:               constants.ClientNameAgave,
			agaveMinVersion:      "invalid-version",
			agaveMaxVersion:      "1.18.5",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              true,
		},
		{
			name:                 "invalid max version format",
			client:               constants.ClientNameAgave,
			agaveMinVersion:      "1.18.0",
			agaveMaxVersion:      "invalid-version",
			firedancerMinVersion: "0.1.0",
			firedancerMaxVersion: "0.1.2",
			wantErr:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Requirements{
				AgaveMinVersion:      tt.agaveMinVersion,
				AgaveMaxVersion:      tt.agaveMaxVersion,
				FiredancerMinVersion: tt.firedancerMinVersion,
				FiredancerMaxVersion: tt.firedancerMaxVersion,
			}

			err := req.SetClient(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetClient() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if req.Client != tt.expectedClient {
					t.Errorf("SetClient() Client = %v, want %v", req.Client, tt.expectedClient)
				}
				if req.HasMinVersion != tt.expectedHasMin {
					t.Errorf("SetClient() HasMinVersion = %v, want %v", req.HasMinVersion, tt.expectedHasMin)
				}
				if req.HasMaxVersion != tt.expectedHasMax {
					t.Errorf("SetClient() HasMaxVersion = %v, want %v", req.HasMaxVersion, tt.expectedHasMax)
				}

				if tt.expectedHasMin && req.MinVersion == nil {
					t.Error("SetClient() MinVersion should be set")
				}
				if tt.expectedHasMax && req.MaxVersion == nil {
					t.Error("SetClient() MaxVersion should be set")
				}

				if tt.expectedHasMin && req.MinVersion != nil {
					if req.MinVersion.String() != tt.expectedMinVersion {
						t.Errorf("SetClient() MinVersion = %v, want %v", req.MinVersion.String(), tt.expectedMinVersion)
					}
				}
				if tt.expectedHasMax && req.MaxVersion != nil {
					if req.MaxVersion.String() != tt.expectedMaxVersion {
						t.Errorf("SetClient() MaxVersion = %v, want %v", req.MaxVersion.String(), tt.expectedMaxVersion)
					}
				}

				// Test constraints string
				if tt.expectedHasMin && tt.expectedHasMax {
					expectedConstraints := ">= " + tt.expectedMinVersion + ",<= " + tt.expectedMaxVersion
					if req.ConstraintsString != expectedConstraints {
						t.Errorf("SetClient() ConstraintsString = %v, want %v", req.ConstraintsString, expectedConstraints)
					}
				} else if tt.expectedHasMin {
					expectedConstraints := ">= " + tt.expectedMinVersion
					if req.ConstraintsString != expectedConstraints {
						t.Errorf("SetClient() ConstraintsString = %v, want %v", req.ConstraintsString, expectedConstraints)
					}
				} else if tt.expectedHasMax {
					expectedConstraints := "<= " + tt.expectedMaxVersion
					if req.ConstraintsString != expectedConstraints {
						t.Errorf("SetClient() ConstraintsString = %v, want %v", req.ConstraintsString, expectedConstraints)
					}
				}

				// Test constraints object
				if req.Constraints == nil {
					t.Error("SetClient() Constraints should be set")
				}
			}
		})
	}
}

func TestRequirements_SetClient_ConstraintsString(t *testing.T) {
	tests := []struct {
		name                string
		client              string
		agaveMinVersion     string
		agaveMaxVersion     string
		expectedConstraints string
		wantErr             bool
	}{
		{
			name:                "min and max versions",
			client:              constants.ClientNameAgave,
			agaveMinVersion:     "1.18.0",
			agaveMaxVersion:     "1.18.5",
			expectedConstraints: ">= 1.18.0,<= 1.18.5",
			wantErr:             false,
		},
		{
			name:                "only min version",
			client:              constants.ClientNameAgave,
			agaveMinVersion:     "1.18.0",
			agaveMaxVersion:     "",
			expectedConstraints: ">= 1.18.0",
			wantErr:             false,
		},
		{
			name:                "only max version",
			client:              constants.ClientNameAgave,
			agaveMinVersion:     "",
			agaveMaxVersion:     "1.18.5",
			expectedConstraints: "<= 1.18.5",
			wantErr:             false,
		},
		{
			name:                "no versions",
			client:              constants.ClientNameAgave,
			agaveMinVersion:     "",
			agaveMaxVersion:     "",
			expectedConstraints: "", // Empty string will cause constraint parsing to fail
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Requirements{
				AgaveMinVersion: tt.agaveMinVersion,
				AgaveMaxVersion: tt.agaveMaxVersion,
			}

			err := req.SetClient(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetClient() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && req.ConstraintsString != tt.expectedConstraints {
				t.Errorf("SetClient() ConstraintsString = %v, want %v", req.ConstraintsString, tt.expectedConstraints)
			}
		})
	}
}
