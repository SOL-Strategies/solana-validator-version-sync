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
	// map of cluster to release notes regex
	releaseNotesRegexes map[string]*regexp.Regexp
	// map of cluster to release title regex
	releaseTitleRegexes map[string]*regexp.Regexp
	repoURL             string
	repoOwner           string
	repoName            string
	clientName          string
	client              *github.Client
	cluster             string
	logger              *log.Logger
	// cachedTagVersions holds all parsed tag versions from the last GetLatestClientVersion call
	cachedTagVersions []*version.Version
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

	// initialize release notes and title regexes
	c.releaseNotesRegexes = make(map[string]*regexp.Regexp)
	c.releaseTitleRegexes = make(map[string]*regexp.Regexp)

	// compile release notes and title regexes for each cluster
	for _, cluster := range constants.ValidClusterNames {
		// compile release notes regexes
		c.releaseNotesRegexes[cluster], err = regexp.Compile(repoConfig.ReleaseNotesRegexes[cluster])
		if err != nil {
			return nil, fmt.Errorf("failed to compile release notes regex: %w", err)
		}
		// compile release title regexes
		c.releaseTitleRegexes[cluster], err = regexp.Compile(repoConfig.ReleaseTitleRegexes[cluster])
		if err != nil {
			return nil, fmt.Errorf("failed to compile release title regex: %w", err)
		}
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

	// map of cluster to version strings
	versionStrings := make(map[string][]string)

	switch c.clientName {
	case constants.ClientNameAgave:
		// agave flag release cluster in release notes
		for _, cluster := range constants.ValidClusterNames {
			versionStrings[cluster] = versionsFromReleaseBodyRegex(releases, c.releaseNotesRegexes[cluster])
		}
	case constants.ClientNameJitoSolana, constants.ClientNameFiredancer:
		// jito-solana and firedancer flags release cluster in release title prefix
		for _, cluster := range constants.ValidClusterNames {
			versionStrings[cluster] = versionsFromReleaseTitleRegex(releases, c.releaseTitleRegexes[cluster])
		}
	}

	// fail if no releases found for client configured cluster
	for cluster, versionStrings := range versionStrings {
		if len(versionStrings) == 0 {
			return nil, fmt.Errorf("no %s releases found matching regex: %s", cluster, c.releaseNotesRegexes[cluster].String())
		}
	}

	// For each cluster, create a versions slice and sort, and get the latest version
	latestClusterVersion := make(map[string]*version.Version)
	c.cachedTagVersions = nil
	for cluster, versionStrings := range versionStrings {
		sortedVersions := c.sortedVersionsFromVersionStrings(versionStrings)
		latestClusterVersion[cluster] = sortedVersions[len(sortedVersions)-1]
		c.cachedTagVersions = append(c.cachedTagVersions, sortedVersions...)
		c.logger.Debug("latest version "+latestClusterVersion[cluster].Core().String(), "client", c.clientName, "cluster", cluster, "repoURL", c.repoURL+"/releases")
	}

	// If cluster is testnet and mainnet version is higher, use mainnet version and warn
	latestVersion = latestClusterVersion[c.cluster]
	if c.cluster == constants.ClusterNameTestnet && latestClusterVersion[constants.ClusterNameMainnetBeta].GreaterThan(latestVersion) {
		latestVersion = latestClusterVersion[constants.ClusterNameMainnetBeta]
		c.logger.Warn(fmt.Sprintf("mainnet v%s > v%s testnet - preferring mainnet version",
			latestClusterVersion[constants.ClusterNameMainnetBeta].Core().String(),
			latestClusterVersion[c.cluster].Core().String()),
			"client", c.clientName, "cluster", c.cluster, "repoURL", c.repoURL+"/releases")
	}

	c.logger.Info("latest version "+latestVersion.Core().String(), "client", c.clientName, "cluster", c.cluster, "repoURL", c.repoURL+"/releases")

	return latestVersion, nil
}

// HasTaggedVersion checks if a tagged version exists in the client repo
func (c *Client) HasTaggedVersion(testVersion *version.Version) (hasTaggedVersion bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// get tags from the client repo and return true if a tag with the version exists
	tags, _, err := c.client.Repositories.ListTags(ctx, c.repoOwner, c.repoName, &github.ListOptions{
		PerPage: 20,
	})
	if err != nil {
		return false, fmt.Errorf("failed to get tags: %w", err)
	}

	// check over the returned tags
	for _, tag := range tags {
		// parse the tag version into a version.Version so we can compare the core versions
		c.logger.Debug("parsing github tag version", "tag", tag.GetName())
		tagVersion, err := version.NewVersion(tag.GetName())
		if err != nil {
			return false, fmt.Errorf("failed to parse tag version: %w", err)
		}

		c.logger.Debug("comparing tag version to test version", "tagVersion", tagVersion.Core().String(), "testVersion", testVersion.Core().String())
		// testVersion exists so return true
		if tagVersion.Core().Compare(testVersion.Core()) == 0 {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) GetRepoURL() string {
	return c.repoURL
}

// NormalizeToTagVersion translates a running version to its equivalent tag version.
//
// Firedancer reports its version differently depending on the source:
//   - GitHub tags use EPOCH.RELEASE.FEATURESET  (e.g. v0.902.40002)
//   - Solana RPC getVersion solana-core field may return EPOCH.RELEASE.0 (e.g. 0.902.0)
//     when the feature-set is not embedded in the version string
//   - After firedancer PR #8945 the binary/RPC may report EPOCH.COMMITCOUNT.FEATURESET
//     (e.g. 0.33670.40002) where COMMITCOUNT differs from RELEASE
//
// To bridge these representations we try two matching strategies in order:
//  1. Feature-set match (PATCH component): works when the running version embeds the
//     feature-set (PATCH > 0), e.g. 0.33670.40002 → v0.902.40002
//  2. MAJOR.MINOR match: works when the running version reports PATCH as 0,
//     e.g. 0.902.0 → v0.902.40002
//
// For all other clients the version is returned unchanged.
func (c *Client) NormalizeToTagVersion(v *version.Version) *version.Version {
	if c.clientName != constants.ClientNameFiredancer {
		return v
	}
	segs := v.Segments()
	if len(segs) < 3 {
		return v
	}

	// Strategy 1: match by feature-set (PATCH) when it is non-zero.
	// Handles the case where the running version embeds the feature-set but uses a
	// different MINOR, e.g. 0.33670.40002 matching tag v0.902.40002.
	featureSet := segs[2]
	if featureSet != 0 {
		for _, tagged := range c.cachedTagVersions {
			tagSegs := tagged.Segments()
			if len(tagSegs) >= 3 && tagSegs[2] == featureSet {
				c.logger.Debug("normalized firedancer running version to tag version (feature-set match)",
					"running", v.Original(), "tag", tagged.Original())
				return tagged
			}
		}
	}

	// Strategy 2: match by MAJOR.MINOR when PATCH is zero (or feature-set match found nothing).
	// Handles the case where the RPC returns EPOCH.RELEASE.0, e.g. 0.902.0 matching tag v0.902.40002.
	for _, tagged := range c.cachedTagVersions {
		tagSegs := tagged.Segments()
		if len(tagSegs) >= 3 && tagSegs[0] == segs[0] && tagSegs[1] == segs[1] {
			c.logger.Debug("normalized firedancer running version to tag version (major.minor match)",
				"running", v.Original(), "tag", tagged.Original())
			return tagged
		}
	}

	c.logger.Warn("could not normalize firedancer running version to tag version - no cached tag matched by feature-set or major.minor",
		"running", v.Original(), "featureSet", featureSet)
	return v
}

// versionsFromReleaseTitleRegex gets versions from non-prerelease releases with titles matching the supplied regex
func versionsFromReleaseTitleRegex(releases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	for _, release := range releases {
		if release.GetPrerelease() {
			log.Debug("skipping github pre-release", "title", release.GetName(), "tag", release.GetTagName())
			continue
		}
		if regex.MatchString(release.GetName()) {
			log.Debug("found matching release", "title", release.GetName(), "tag", release.GetTagName(), "version", release.GetTagName())
			versionStrings = append(versionStrings, release.GetTagName())
		}
	}
	return versionStrings
}

// versionsFromReleaseBodyRegex gets versions from non-prerelease releases with bodies matching the supplied regex
func versionsFromReleaseBodyRegex(releases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	for _, release := range releases {
		if release.GetPrerelease() {
			log.Debug("skipping github pre-release", "title", release.GetName(), "tag", release.GetTagName())
			continue
		}
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

func (c *Client) sortedVersionsFromVersionStrings(versionStrings []string) (sortedVersions []*version.Version) {
	c.logger.Debug("sorting versions", "versionStrings", versionStrings)
	sortedVersions = make([]*version.Version, len(versionStrings))
	for i, raw := range versionStrings {
		v, _ := version.NewVersion(raw)
		sortedVersions[i] = v
	}
	sort.Sort(version.Collection(sortedVersions))
	c.logger.Debug("sorted versions", "sortedVersions", sortedVersions)
	return sortedVersions
}
