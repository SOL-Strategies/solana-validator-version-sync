package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	url := "http://localhost:8899"
	client := NewClient(url)

	if client == nil {
		t.Error("NewClient() returned nil")
	}
	if client.url != url {
		t.Errorf("NewClient() url = %v, want %v", client.url, url)
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

func TestJSONRPCRequest_StructFields(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getVersion",
		Params:  []interface{}{},
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC to be 2.0, got %s", req.JSONRPC)
	}
	if req.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", req.ID)
	}
	if req.Method != "getVersion" {
		t.Errorf("Expected Method to be getVersion, got %s", req.Method)
	}
	if len(req.Params) != 0 {
		t.Errorf("Expected Params to be empty, got %v", req.Params)
	}
}

func TestJSONRPCResponse_StructFields(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"version": "1.18.0"},
		Error:   nil,
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC to be 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", resp.ID)
	}
	if resp.Result == nil {
		t.Error("Expected Result to be set")
	}
	if resp.Error != nil {
		t.Error("Expected Error to be nil")
	}
}

func TestRPCError_StructFields(t *testing.T) {
	rpcErr := RPCError{
		Code:    -32601,
		Message: "Method not found",
	}

	if rpcErr.Code != -32601 {
		t.Errorf("Expected Code to be -32601, got %d", rpcErr.Code)
	}
	if rpcErr.Message != "Method not found" {
		t.Errorf("Expected Message to be 'Method not found', got %s", rpcErr.Message)
	}
}

func TestValidatorState_StructFields(t *testing.T) {
	state := ValidatorState{
		RunningVersion: "1.18.0",
		IdentityPubkey: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
		Role:           "active",
	}

	if state.RunningVersion != "1.18.0" {
		t.Errorf("Expected RunningVersion to be 1.18.0, got %s", state.RunningVersion)
	}
	if state.IdentityPubkey != "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM" {
		t.Errorf("Expected IdentityPubkey to be 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM, got %s", state.IdentityPubkey)
	}
	if state.Role != "active" {
		t.Errorf("Expected Role to be active, got %s", state.Role)
	}
}

func TestClient_makeRPCCall(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse JSONRPCResponse
		serverStatus   int
		wantErr        bool
	}{
		{
			name: "successful RPC call",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{"version": "1.18.0"},
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name: "RPC error response",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			},
			serverStatus: http.StatusOK,
			wantErr:      true,
		},
		{
			name: "HTTP error status",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{"version": "1.18.0"},
			},
			serverStatus: http.StatusInternalServerError,
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

			client := NewClient(server.URL)
			ctx := context.Background()

			resp, err := client.makeRPCCall(ctx, "getVersion", []interface{}{})
			if (err != nil) != tt.wantErr {
				t.Errorf("makeRPCCall() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && resp == nil {
				t.Error("makeRPCCall() returned nil response")
			}
		})
	}
}

func TestClient_getIdentity(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse JSONRPCResponse
		wantIdentity   string
		wantErr        bool
	}{
		{
			name: "successful identity call",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]interface{}{
					"identity": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
				},
			},
			wantIdentity: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			wantErr:      false,
		},
		{
			name: "invalid response format",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  "invalid format",
			},
			wantErr: true,
		},
		{
			name: "missing identity field",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]interface{}{
					"version": "1.18.0",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			ctx := context.Background()

			identity, err := client.getIdentity(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getIdentity() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && identity != tt.wantIdentity {
				t.Errorf("getIdentity() = %v, want %v", identity, tt.wantIdentity)
			}
		})
	}
}

func TestClient_getVersion(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse JSONRPCResponse
		wantVersion    string
		wantErr        bool
	}{
		{
			name: "successful version call",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]interface{}{
					"solana-core": "1.18.0",
				},
			},
			wantVersion: "1.18.0",
			wantErr:     false,
		},
		{
			name: "invalid response format",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  "invalid format",
			},
			wantErr: true,
		},
		{
			name: "missing solana-core field",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]interface{}{
					"version": "1.18.0",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			ctx := context.Background()

			version, err := client.getVersion(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && version != tt.wantVersion {
				t.Errorf("getVersion() = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}

func TestClient_getHealth(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse JSONRPCResponse
		wantHealth     string
		wantErr        bool
	}{
		{
			name: "successful health call",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  "ok",
			},
			wantHealth: "ok",
			wantErr:    false,
		},
		{
			name: "invalid response format",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{"health": "ok"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			ctx := context.Background()

			health, err := client.getHealth(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getHealth() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && health != tt.wantHealth {
				t.Errorf("getHealth() = %v, want %v", health, tt.wantHealth)
			}
		})
	}
}

func TestClient_GetIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result: map[string]interface{}{
				"identity": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	identity, err := client.GetIdentity()

	if err != nil {
		t.Errorf("GetIdentity() error = %v", err)
	}
	if identity != "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM" {
		t.Errorf("GetIdentity() = %v, want %v", identity, "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM")
	}
}

func TestClient_GetVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result: map[string]interface{}{
				"solana-core": "1.18.0",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	version, err := client.GetVersion()

	if err != nil {
		t.Errorf("GetVersion() error = %v", err)
	}
	if version != "1.18.0" {
		t.Errorf("GetVersion() = %v, want %v", version, "1.18.0")
	}
}

func TestClient_GetHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  "ok",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	health, err := client.GetHealth()

	if err != nil {
		t.Errorf("GetHealth() error = %v", err)
	}
	if health != "ok" {
		t.Errorf("GetHealth() = %v, want %v", health, "ok")
	}
}

func TestClient_Timeout(t *testing.T) {
	// Create a server that takes longer than the client timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than 30s timeout
		json.NewEncoder(w).Encode(JSONRPCResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetHealth()

	if err == nil {
		t.Error("GetHealth() should have timed out")
	}
}

func TestClient_getClusterNodes(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse JSONRPCResponse
		wantNodes      int
		wantErr        bool
	}{
		{
			name: "successful cluster nodes call",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: []interface{}{
					map[string]interface{}{
						"gossip": "127.0.0.1:8001",
						"pubkey": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
					},
					map[string]interface{}{
						"gossip": "127.0.0.1:8002",
						"pubkey": "AnotherKey123456789012345678901234567890",
					},
				},
			},
			wantNodes: 2,
			wantErr:   false,
		},
		{
			name: "empty cluster nodes",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  []interface{}{},
			},
			wantNodes: 0,
			wantErr:   false,
		},
		{
			name: "RPC error response",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid response format",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  "invalid format",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			ctx := context.Background()

			nodes, err := client.getClusterNodes(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getClusterNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if nodes == nil {
					t.Error("getClusterNodes() returned nil nodes")
					return
				}
				if len(*nodes) != tt.wantNodes {
					t.Errorf("getClusterNodes() returned %d nodes, want %d", len(*nodes), tt.wantNodes)
				}
			}
		})
	}
}

func TestClient_GetNodeWithIdentityPublicKey(t *testing.T) {
	tests := []struct {
		name              string
		serverResponse    JSONRPCResponse
		identityPublicKey string
		wantFound         bool
		wantNodePubkey    string
		wantNodeGossip    string
		wantErr           bool
	}{
		{
			name: "node found",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: []interface{}{
					map[string]interface{}{
						"gossip": "127.0.0.1:8001",
						"pubkey": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
					},
					map[string]interface{}{
						"gossip": "127.0.0.1:8002",
						"pubkey": "AnotherKey123456789012345678901234567890",
					},
				},
			},
			identityPublicKey: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			wantFound:         true,
			wantNodePubkey:    "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			wantNodeGossip:    "127.0.0.1:8001",
			wantErr:           false,
		},
		{
			name: "node not found",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result: []interface{}{
					map[string]interface{}{
						"gossip": "127.0.0.1:8001",
						"pubkey": "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
					},
				},
			},
			identityPublicKey: "NotFoundKey123456789012345678901234567890",
			wantFound:         false,
			wantErr:           false, // Node not found is not an error, just not found
		},
		{
			name: "empty cluster nodes",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  []interface{}{},
			},
			identityPublicKey: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			wantFound:         false,
			wantErr:           false, // Empty result is not an error, just not found
		},
		{
			name: "RPC error getting cluster nodes",
			serverResponse: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &RPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			},
			identityPublicKey: "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
			wantFound:         false,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := NewClient(server.URL)

			found, node, err := client.GetNodeWithIdentityPublicKey(tt.identityPublicKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeWithIdentityPublicKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if found != tt.wantFound {
				t.Errorf("GetNodeWithIdentityPublicKey() found = %v, want %v", found, tt.wantFound)
			}

			if tt.wantFound && node != nil {
				if node.Pubkey != tt.wantNodePubkey {
					t.Errorf("GetNodeWithIdentityPublicKey() node.Pubkey = %v, want %v", node.Pubkey, tt.wantNodePubkey)
				}
				if node.Gossip != tt.wantNodeGossip {
					t.Errorf("GetNodeWithIdentityPublicKey() node.Gossip = %v, want %v", node.Gossip, tt.wantNodeGossip)
				}
			}
		})
	}
}
