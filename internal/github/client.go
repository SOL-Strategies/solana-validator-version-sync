package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// Client represents a GitHub API client
type Client struct {
	baseURL string
	client  *http.Client
	logger  *log.Logger
}

// NewClient creates a new GitHub client
func NewClient() *Client {
	return &Client{
		baseURL: "https://api.github.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.WithPrefix("github"),
	}
}

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
}

// GetAvailableVersions gets available versions from GitHub releases that match the given regex
func (c *Client) GetAvailableVersions(repoURL, releaseNotesRegex string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Extract owner and repo from URL
	owner, repo, err := c.extractOwnerRepo(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract owner/repo from URL: %w", err)
	}

	// Compile the regex
	regex, err := regexp.Compile(releaseNotesRegex)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	// Get releases from GitHub API
	releases, err := c.getReleases(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get releases: %w", err)
	}

	// Filter releases that match the regex
	var matchingVersions []string
	for _, release := range releases {
		// Skip drafts and prereleases
		if release.Draft || release.Prerelease {
			continue
		}

		// Check if the release body matches the regex
		if regex.MatchString(release.Body) {
			// Extract version from tag name (remove 'v' prefix if present)
			version := strings.TrimPrefix(release.TagName, "v")
			matchingVersions = append(matchingVersions, version)
		}
	}

	// Sort versions (this is a simple string sort, might need more sophisticated version comparison)
	sort.Strings(matchingVersions)

	return matchingVersions, nil
}

// extractOwnerRepo extracts owner and repo from a GitHub URL
func (c *Client) extractOwnerRepo(url string) (string, string, error) {
	// Handle different GitHub URL formats
	// https://github.com/owner/repo
	// git@github.com:owner/repo.git

	url = strings.TrimSuffix(url, ".git")

	var parts []string
	if strings.Contains(url, "github.com/") {
		parts = strings.Split(url, "github.com/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
		}
		pathParts := strings.Split(parts[1], "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
		}
		return pathParts[0], pathParts[1], nil
	} else if strings.Contains(url, "github.com:") {
		parts = strings.Split(url, "github.com:")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
		}
		pathParts := strings.Split(parts[1], "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL format: %s", url)
		}
		return pathParts[0], pathParts[1], nil
	}

	return "", "", fmt.Errorf("unsupported GitHub URL format: %s", url)
}

// getReleases gets releases from GitHub API
func (c *Client) getReleases(ctx context.Context, owner, repo string) ([]Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers for better API usage
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "solana-validator-version-sync")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return releases, nil
}
