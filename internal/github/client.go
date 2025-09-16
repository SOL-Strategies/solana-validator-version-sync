package github

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/go-version"
	"github.com/sol-strategies/solana-validator-version-sync/internal/constants"
)

var (
	// Handle different GitHub URL formats:
	// https://github.com/owner/repo
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	// git@github.com:owner/repo
	// Regex pattern to match both HTTPS and SSH GitHub URLs
	// Group 1: owner, Group 2: repo (without .git suffix)
	githubRepoAndOwnerFromURLRegex = regexp.MustCompile(`(?:https://github\.com/|git@github\.com:)([^/]+)/([^/]+?)(?:\.git)?$`)
)

// Client represents a GitHub API client
type Client struct {
	releaseNotesRegex *regexp.Regexp
	releaseTitleRegex *regexp.Regexp
	repoURL           string
	repoOwner         string
	repoName          string
	clientName        string
	client            *github.Client
	cluster           string
	logger            *log.Logger
}

// Options represents the options for creating a new GitHub client
type Options struct {
	Cluster string
	Client  string
}

// NewClient creates a new GitHub client
func NewClient(opts Options) (c *Client, err error) {
	// Get client repo config
	repoConfig, ok := clientRepoConfigs[opts.Client]
	if !ok {
		return nil, fmt.Errorf("client repo config not found for client: %s", opts.Client)
	}

	c = &Client{
		cluster:    opts.Cluster,
		clientName: opts.Client,
		repoURL:    repoConfig.URL,
		client:     github.NewClient(nil), // No auth token for public repos
		logger:     log.WithPrefix("github"),
	}

	// extract owner and repo from URL
	err = c.setOwnerAndRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to extract owner/repo from URL: %w", err)
	}

	// compile release notes regex
	c.releaseNotesRegex, err = regexp.Compile(repoConfig.ReleaseNotesRegexes[c.cluster])
	if err != nil {
		return nil, fmt.Errorf("failed to compile release notes regex: %w", err)
	}

	// compile release title regex
	c.releaseTitleRegex, err = regexp.Compile(repoConfig.ReleaseTitleRegexes[c.cluster])
	if err != nil {
		return nil, fmt.Errorf("failed to compile release title regex: %w", err)
	}

	return c, nil
}

// GetLatestClientVersion gets the latest version from GitHub releases that match the given notes regex for the cluster and client
func (c *Client) GetLatestClientVersion() (latestVersion *version.Version, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get releases from GitHub API using go-github
	releases, _, err := c.client.Repositories.ListReleases(ctx, c.repoOwner, c.repoName, &github.ListOptions{
		PerPage: 20, // We just need the last few releases
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get releases: %w", err)
	}

	versionStrings := []string{}

	switch c.clientName {
	case constants.ClientNameAgave:
		// agave flag release cluster in release notes
		versionStrings = versionsFromReleaseBodyRegex(releases, c.releaseNotesRegex)
	case constants.ClientNameJitoSolana, constants.ClientNameFiredancer:
		// jito-solana and firedancer flags release cluster in release title prefix
		versionStrings = versionsFromReleaseTitleRegex(releases, c.releaseTitleRegex)
	}

	if len(versionStrings) == 0 {
		return nil, fmt.Errorf("no releases found matching regex: %s", c.releaseNotesRegex.String())
	}

	// Create versions slice and sort
	versions := make([]*version.Version, len(versionStrings))
	for i, raw := range versionStrings {
		v, _ := version.NewVersion(raw)
		versions[i] = v
	}
	sort.Sort(version.Collection(versions))

	latestVersion = versions[len(versions)-1]

	c.logger.Info("latest version "+latestVersion.Core().String(), "client", c.clientName, "cluster", c.cluster, "repoURL", c.repoURL+"/releases")

	return latestVersion, nil
}

// versionsFromReleaseTitleRegex gets versions from releases with titles matching the supplied regex
func versionsFromReleaseTitleRegex(releases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	for _, release := range releases {
		if regex.MatchString(release.GetName()) {
			log.Debug("found matching release", "title", release.GetName(), "tag", release.GetTagName(), "version", release.GetTagName())
			versionStrings = append(versionStrings, release.GetTagName())
		}
	}
	return versionStrings
}

// versionsFromReleaseBodyRegex gets versions from releases with bodies matching the supplied regex
func versionsFromReleaseBodyRegex(releases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	for _, release := range releases {
		if regex.MatchString(release.GetBody()) {
			versionStrings = append(versionStrings, release.GetTagName())
		}
	}
	return versionStrings
}

// setOwnerAndRepo extracts owner and repo from a GitHub URL
func (c *Client) setOwnerAndRepo() (err error) {
	matches := githubRepoAndOwnerFromURLRegex.FindStringSubmatch(c.repoURL)
	if len(matches) != 3 {
		return fmt.Errorf("unsupported GitHub URL format: %s", c.repoURL)
	}

	c.repoOwner = matches[1]
	c.repoName = matches[2]

	return nil
}
