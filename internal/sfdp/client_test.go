package sfdp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

func TestNewClient(t *testing.T) {
	opts := Options{
		Cluster: "mainnet-beta",
		Client:  constants.ClientNameAgave,
	}
	client := NewClient(opts)

	if client == nil {
		t.Error("NewClient() returned nil")
	}
	if client.baseURL != "https://api.solana.org/api" {
		t.Errorf("NewClient() baseURL = %v, want %v", client.baseURL, "https://api.solana.org/api")
	}
	if client.cluster != "mainnet-beta" {
		t.Errorf("NewClient() cluster = %v, want %v", client.cluster, "mainnet-beta")
	}
	if client.clientName != constants.ClientNameAgave {
		t.Errorf("NewClient() clientName = %v, want %v", client.clientName, constants.ClientNameAgave)
	}
	if client.client == nil {
		t.Error("NewClient() should initialize HTTP client")
	}
	if client.logger == nil {
		t.Error("NewClient() should initialize logger")
	}
	if client.client.Timeout != 30*time.Second {
		t.Errorf("NewClient() timeout = %v, want %v", client.client.Timeout, 30*time.Second)
	}
}

func TestOptions_StructFields(t *testing.T) {
	opts := Options{
		Cluster: "testnet",
		Client:  constants.ClientNameFiredancer,
	}

	if opts.Cluster != "testnet" {
		t.Errorf("Expected Cluster to be testnet, got %s", opts.Cluster)
	}
	if opts.Client != constants.ClientNameFiredancer {
		t.Errorf("Expected Client to be %s, got %s", constants.ClientNameFiredancer, opts.Client)
	}
}

func TestRequirementsResponse_StructFields(t *testing.T) {
	resp := RequirementsResponse{
		Error: "",
		Data: []Requirements{
			{
				Epoch:   500,
				Cluster: "mainnet-beta",
			},
		},
	}

	if resp.Error != "" {
		t.Errorf("Expected Error to be empty, got %s", resp.Error)
	}
	if len(resp.Data) != 1 {
		t.Errorf("Expected Data length to be 1, got %d", len(resp.Data))
	}
	if resp.Data[0].Epoch != 500 {
		t.Errorf("Expected first requirement epoch to be 500, got %d", resp.Data[0].Epoch)
	}
}

func TestClient_GetLatestRequirements(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse RequirementsResponse
		serverStatus   int
		clientName     string
		wantErr        bool
		expectedEpoch  int
	}{
		{
			name: "successful requirements call with single requirement",
			serverResponse: RequirementsResponse{
				Data: []Requirements{
					{
						Epoch:                500,
						Cluster:              "mainnet-beta",
						AgaveMinVersion:      "1.18.0",
						AgaveMaxVersion:      "1.18.5",
						FiredancerMinVersion: "0.1.0",
						FiredancerMaxVersion: "0.1.2",
					},
				},
			},
			serverStatus:  http.StatusOK,
			clientName:    constants.ClientNameAgave,
			wantErr:       false,
			expectedEpoch: 500,
		},
		{
			name: "successful requirements call with multiple requirements",
			serverResponse: RequirementsResponse{
				Data: []Requirements{
					{
						Epoch:                500,
						Cluster:              "mainnet-beta",
						AgaveMinVersion:      "1.18.0",
						AgaveMaxVersion:      "1.18.5",
						FiredancerMinVersion: "0.1.0",
						FiredancerMaxVersion: "0.1.2",
					},
					{
						Epoch:                501,
						Cluster:              "mainnet-beta",
						AgaveMinVersion:      "1.18.1",
						AgaveMaxVersion:      "1.18.6",
						FiredancerMinVersion: "0.1.1",
						FiredancerMaxVersion: "0.1.3",
					},
				},
			},
			serverStatus:  http.StatusOK,
			clientName:    constants.ClientNameFiredancer,
			wantErr:       false,
			expectedEpoch: 501, // Should pick the highest epoch
		},
		{
			name: "SFDP API error response",
			serverResponse: RequirementsResponse{
				Error: "Invalid cluster",
				Data:  []Requirements{},
			},
			serverStatus: http.StatusOK,
			clientName:   constants.ClientNameAgave,
			wantErr:      true,
		},
		{
			name: "HTTP error status",
			serverResponse: RequirementsResponse{
				Data: []Requirements{
					{
						Epoch:   500,
						Cluster: "mainnet-beta",
					},
				},
			},
			serverStatus: http.StatusInternalServerError,
			clientName:   constants.ClientNameAgave,
			wantErr:      true,
		},
		{
			name: "no requirements data",
			serverResponse: RequirementsResponse{
				Data: []Requirements{},
			},
			serverStatus: http.StatusOK,
			clientName:   constants.ClientNameAgave,
			wantErr:      true,
		},
		{
			name: "invalid client name",
			serverResponse: RequirementsResponse{
				Data: []Requirements{
					{
						Epoch:   500,
						Cluster: "mainnet-beta",
					},
				},
			},
			serverStatus: http.StatusOK,
			clientName:   "invalid-client",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			// Override the baseURL for testing
			opts := Options{
				Cluster: "mainnet-beta",
				Client:  tt.clientName,
			}
			client := NewClient(opts)
			client.baseURL = server.URL

			requirements, err := client.GetLatestRequirements()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestRequirements() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if requirements == nil {
					t.Error("GetLatestRequirements() returned nil requirements")
				} else if requirements.Epoch != tt.expectedEpoch {
					t.Errorf("GetLatestRequirements() epoch = %v, want %v", requirements.Epoch, tt.expectedEpoch)
				}
			}
		})
	}
}

func TestClient_GetLatestRequirements_URL(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		json.NewEncoder(w).Encode(RequirementsResponse{
			Data: []Requirements{
				{
					Epoch:           500,
					Cluster:         "mainnet-beta",
					AgaveMinVersion: "1.18.0",
					AgaveMaxVersion: "1.18.5",
				},
			},
		})
	}))
	defer server.Close()

	opts := Options{
		Cluster: "testnet",
		Client:  constants.ClientNameAgave,
	}
	client := NewClient(opts)
	client.baseURL = server.URL + "/api"

	_, err := client.GetLatestRequirements()
	if err != nil {
		t.Errorf("GetLatestRequirements() error = %v", err)
	}

	expectedURL := "/api/epoch/required_versions?cluster=testnet"
	if capturedURL != expectedURL {
		t.Errorf("GetLatestRequirements() URL = %v, want %v", capturedURL, expectedURL)
	}
}
