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
	baseURL    string
	cluster    string
	clientName string
	client     *http.Client
	logger     *log.Logger
}

// Options represents the options for creating a new SFDP client
type Options struct {
	Cluster string
	Client  string
}

// NewClient creates a new SFDP client
func NewClient(opts Options) *Client {
	return &Client{
		baseURL:    "https://api.solana.org/api",
		cluster:    opts.Cluster,
		clientName: opts.Client,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.WithPrefix("sfdp"),
	}
}

// RequirementsResponse represents the response from the SFDP API
type RequirementsResponse struct {
	Error string         `json:"error,omitempty"`
	Data  []Requirements `json:"data"`
}

// GetLatestRequirements gets version requirements from SFDP for a given cluster
func (c *Client) GetLatestRequirements() (latestRequirements *Requirements, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/epoch/required_versions?cluster=%s", c.baseURL, c.cluster)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	var result RequirementsResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("SFDP API error: %s", result.Error)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no requirements data found")
	}

	// Get the latest requirements (item in the slice with the highest epoch number)
	latestRequirements = &result.Data[0]
	for _, requirement := range result.Data {
		if requirement.Epoch > latestRequirements.Epoch {
			latestRequirements = &requirement
		}
	}

	c.logger.Debug("latest requirements", "requirements", latestRequirements, "epoch", latestRequirements.Epoch)

	// set the client
	err = latestRequirements.SetClient(c.clientName)
	if err != nil {
		return nil, fmt.Errorf("failed to set client: %w", err)
	}

	return latestRequirements, nil
}
