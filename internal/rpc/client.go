package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Client represents an RPC client for communicating with the validator
type Client struct {
	url    string
	client *http.Client
	logger *log.Logger
}

// NewClient creates a new RPC client
func NewClient(url string) *Client {
	return &Client{
		url: url,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.WithPrefix("rpc"),
	}
}

// ValidatorState represents the current state of the validator
type ValidatorState struct {
	// RunningVersion is the currently running version of the validator
	RunningVersion string
	// IdentityPubkey is the public key of the validator's identity
	IdentityPubkey string
	// Role is the role of the validator (active/passive)
	Role string
}

// GetValidatorState gets the current state of the validator
func (c *Client) GetValidatorState() (*ValidatorState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the validator's identity
	identity, err := c.getIdentity(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator identity: %w", err)
	}

	// Get the validator's version
	version, err := c.getVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator version: %w", err)
	}

	// Get the validator's role (this would need to be determined based on gossip or other means)
	role, err := c.getRole(ctx, identity)
	if err != nil {
		c.logger.Warn("failed to determine validator role", "error", err)
		role = "unknown"
	}

	return &ValidatorState{
		RunningVersion: version,
		IdentityPubkey: identity,
		Role:           role,
	}, nil
}

// makeRPCCall makes a JSON-RPC call to the validator
func (c *Client) makeRPCCall(ctx context.Context, method string, params []interface{}) (*JSONRPCResponse, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

// getIdentity gets the validator's identity public key
func (c *Client) getIdentity(ctx context.Context) (string, error) {
	resp, err := c.makeRPCCall(ctx, "getIdentity", []interface{}{})
	if err != nil {
		return "", fmt.Errorf("failed to get identity: %w", err)
	}

	// Extract the value from the result
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	value, ok := result["value"].(string)
	if !ok {
		return "", fmt.Errorf("invalid value format")
	}

	return value, nil
}

// getVersion gets the validator's version
func (c *Client) getVersion(ctx context.Context) (string, error) {
	resp, err := c.makeRPCCall(ctx, "getVersion", []interface{}{})
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	// Extract the solana-core version from the result
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	version, ok := result["solana-core"].(string)
	if !ok {
		return "", fmt.Errorf("invalid version format")
	}

	return version, nil
}

// getRole determines the validator's role (active/passive)
// This is a simplified implementation - in reality, this would need to check gossip
// or other mechanisms to determine if the validator is running as active or passive
func (c *Client) getRole(ctx context.Context, identity string) (string, error) {
	// For now, return a mock role - this would be implemented with actual RPC calls
	return "active", nil
}

// Health checks if the validator is healthy
func (c *Client) Health() error {
	// Make a simple HTTP request to the health endpoint
	resp, err := http.Get(c.url + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}
