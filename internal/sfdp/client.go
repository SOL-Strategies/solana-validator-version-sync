package sfdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

// Client represents an SFDP API client
type Client struct {
	baseURL string
	client  *http.Client
	logger  *log.Logger
}

// NewClient creates a new SFDP client
func NewClient() *Client {
	return &Client{
		baseURL: "https://api.solana.org/api",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.WithPrefix("sfdp"),
	}
}

// Validator represents a validator in the SFDP
type Validator struct {
	State string `json:"state"`
	// Add other fields as needed
}

// Requirements represents SFDP version requirements
type Requirements struct {
	MinVersion string `json:"min_version"`
	MaxVersion string `json:"max_version"`
}

// GetValidator gets validator information from SFDP
func (c *Client) GetValidator(identityPubkey string) (*Validator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/validators/%s", c.baseURL, identityPubkey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("validator not found in SFDP")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SFDP API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Error   string `json:"error,omitempty"`
		Message string `json:"message,omitempty"`
		State   string `json:"state,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("SFDP API error: %s", result.Message)
	}

	return &Validator{
		State: result.State,
	}, nil
}

// GetRequirements gets version requirements from SFDP for a given cluster
func (c *Client) GetRequirements(cluster string) (*Requirements, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/epoch/required_versions?cluster=%s", c.baseURL, cluster)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SFDP API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Error string `json:"error,omitempty"`
		Data  []struct {
			AgaveMinVersion string `json:"agave_min_version"`
			AgaveMaxVersion string `json:"agave_max_version"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("SFDP API error: %s", result.Error)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no requirements data found")
	}

	// Get the latest requirements (last item in the array)
	latest := result.Data[len(result.Data)-1]

	return &Requirements{
		MinVersion: latest.AgaveMinVersion,
		MaxVersion: latest.AgaveMaxVersion,
	}, nil
}
