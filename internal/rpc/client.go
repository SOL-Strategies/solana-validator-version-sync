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

// clusterNode represents a node in the cluster
type clusterNodeResult struct {
	Gossip string `json:"gossip"`
	Pubkey string `json:"pubkey"`
}

type clusterNodeResults []clusterNodeResult

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

	c.logger.Debug("identity response", "result", resp.Result)

	identity, ok := result["identity"].(string)
	if !ok {
		return "", fmt.Errorf("invalid identity format")
	}

	return identity, nil
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

// getHealth gets the validator's health
func (c *Client) getHealth(ctx context.Context) (string, error) {
	resp, err := c.makeRPCCall(ctx, "getHealth", []interface{}{})
	if err != nil {
		return "", fmt.Errorf("failed to get health: %w", err)
	}

	// Extract the solana-core version from the result
	result, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	return result, nil
}

// getClusterNodes gets all delinquent and non-delinquent validators from gossip
func (c *Client) getClusterNodes(ctx context.Context) (*clusterNodeResults, error) {
	resp, err := c.makeRPCCall(ctx, "getClusterNodes", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// turn the result array into a clusterNodeResults
	resultArray, ok := resp.Result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: expected array, got %T", resp.Result)
	}

	clusterNodeResults := clusterNodeResults{}
	for _, item := range resultArray {
		nodeMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid node format: expected map, got %T", item)
		}

		node := clusterNodeResult{}
		if gossip, ok := nodeMap["gossip"].(string); ok {
			node.Gossip = gossip
		}
		if pubkey, ok := nodeMap["pubkey"].(string); ok {
			node.Pubkey = pubkey
		}
		clusterNodeResults = append(clusterNodeResults, node)
	}
	return &clusterNodeResults, nil
}

// Health checks if the validator is healthy
func (c *Client) GetHealth() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.getHealth(ctx)
}

// GetVersion gets the validator's version (public method)
func (c *Client) GetVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.getVersion(ctx)
}

// GetIdentity gets the validator's identity public key (public method)
func (c *Client) GetIdentity() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.getIdentity(ctx)
}

// GetNodeWithIdentityPublicKey gets a validator with the given identity public key
func (c *Client) GetNodeWithIdentityPublicKey(identityPublicKey string) (found bool, node *clusterNodeResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clusterNodes, err := c.getClusterNodes(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	for _, n := range *clusterNodes {
		if n.Pubkey == identityPublicKey {
			return true, &n, nil
		}
	}
	// Node not found, but this is not an error - we successfully queried gossip
	return false, nil, nil
}
