package github

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

	// jitoVersionSuffixRegex matches the -jito[.N] suffix in jito-solana git tags
	// (e.g. v4.0.0-beta.2-jito, v3.1.10-jito.1). The RPC does not include this suffix.
	jitoVersionSuffixRegex = regexp.MustCompile(`-jito(\.\d+)?$`)

	// ErrNoMatchingTaggedVersion indicates the client repo does not currently have an
	// eligible tag for the configured cluster. Callers may treat this as a soft skip.
	ErrNoMatchingTaggedVersion = errors.New("no matching tagged version available")
)

// Client represents a GitHub API client
type Client struct {
	// map of cluster to release notes regex
	releaseNotesRegexes map[string]*regexp.Regexp
	// map of cluster to release title regex
	releaseTitleRegexes map[string]*regexp.Regexp
	// map of cluster to git tag regex
	tagRegexes map[string]*regexp.Regexp
	repoURL    string
	repoOwner  string
	repoName   string
	clientName string
	client     *github.Client
	cluster    string
	logger     *log.Logger
	// cachedTagVersions holds all parsed tag versions from the last GetLatestClientVersion call
	cachedTagVersions []*version.Version
	cachedTagInfos    []tagVersionInfo
}

type tagVersionInfo struct {
	TagName     string
	Version     *version.Version
	TestnetOnly bool
}

// Options represents the options for creating a new GitHub client
type Options struct {
	Cluster string
	Client  string
}

// NewClient creates a new GitHub client
func NewClient(opts Options) (c *Client, err error) {
	normalizedClient := constants.NormalizeClientName(opts.Client)

	// Get client repo config
	repoConfig, ok := clientRepoConfigs[normalizedClient]
	if !ok {
		return nil, fmt.Errorf("client repo config not found for client: %s", opts.Client)
	}

	c = &Client{
		cluster:    opts.Cluster,
		clientName: normalizedClient,
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
	c.tagRegexes = make(map[string]*regexp.Regexp)

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
		// compile tag regexes
		c.tagRegexes[cluster], err = regexp.Compile(repoConfig.TagRegexes[cluster])
		if err != nil {
			return nil, fmt.Errorf("failed to compile tag regex: %w", err)
		}
	}
	return c, nil
}

// GetLatestClientVersion gets the latest version from GitHub releases that match the given notes regex for the cluster and client
func (c *Client) GetLatestClientVersion() (latestVersion *version.Version, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch c.clientName {
	case constants.ClientNameAgave:
		// Get releases from GitHub API using go-github
		releases, _, err := c.client.Repositories.ListReleases(ctx, c.repoOwner, c.repoName, &github.ListOptions{
			PerPage: 20, // We just need the last few releases
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get releases: %w", err)
		}
		// map of cluster to version strings
		versionStrings := make(map[string][]string)
		// agave flag release cluster in release notes
		for _, cluster := range constants.ValidClusterNames {
			versionStrings[cluster] = versionsFromReleaseBodyRegexWithPrerelease(releases, c.releaseNotesRegexes[cluster], true)
		}
		return c.latestVersionFromClusterVersionStrings(versionStrings)
	case constants.ClientNameJitoSolana:
		return c.getLatestJitoSolanaVersion(ctx)
	case constants.ClientNameFiredancer:
		releases, _, err := c.client.Repositories.ListReleases(ctx, c.repoOwner, c.repoName, &github.ListOptions{
			PerPage: 20, // We just need the last few releases
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get releases: %w", err)
		}
		return c.latestVersionFromClusterVersionStrings(c.firedancerVersionStringsByCluster(releases))
	case constants.ClientNameRakurai:
		return c.getLatestRakuraiVersion(ctx)
	default:
		return nil, fmt.Errorf("unsupported client: %s", c.clientName)
	}
}

func (c *Client) firedancerVersionStringsByCluster(releases []*github.RepositoryRelease) map[string][]string {
	versionStrings := make(map[string][]string)
	// Firedancer usually flags release cluster in the release title prefix.
	for _, cluster := range constants.ValidClusterNames {
		versionStrings[cluster] = versionsFromReleaseTitleRegexWithPrerelease(releases, c.releaseTitleRegexes[cluster], cluster == constants.ClusterNameTestnet)
	}

	// Some Testnet-titled Frankendancer releases are explicitly suitable for
	// limited mainnet use in the notes. Treat only those as mainnet candidates.
	testnetTitleRegex := c.releaseTitleRegexes[constants.ClusterNameTestnet]
	mainnetNotesRegex := c.releaseNotesRegexes[constants.ClusterNameMainnetBeta]
	if testnetTitleRegex != nil && mainnetNotesRegex != nil {
		versionStrings[constants.ClusterNameMainnetBeta] = appendUniqueVersionStrings(
			versionStrings[constants.ClusterNameMainnetBeta],
			versionsFromReleaseTitleAndBodyRegexWithPrerelease(releases, testnetTitleRegex, mainnetNotesRegex, true)...,
		)
	}

	return versionStrings
}

func (c *Client) getLatestJitoSolanaVersion(ctx context.Context) (latestVersion *version.Version, err error) {
	jitoReleases, _, err := c.client.Repositories.ListReleases(ctx, c.repoOwner, c.repoName, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get jito-solana releases: %w", err)
	}

	agaveOwner, agaveRepo, err := ownerAndRepoFromURL(clientRepoConfigs[constants.ClientNameAgave].URL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract agave owner/repo from URL: %w", err)
	}

	agaveReleases, _, err := c.client.Repositories.ListReleases(ctx, agaveOwner, agaveRepo, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get agave releases for jito-solana classification: %w", err)
	}

	versionStrings := make(map[string][]string)
	// jito-solana tags are Agave versions with a -jito suffix. Classify the
	// underlying Agave version from Agave release notes, then map back to the
	// matching Jito release tag so title prefixes are not required.
	for _, cluster := range constants.ValidClusterNames {
		agaveReleaseNotesRegex, err := regexp.Compile(clientRepoConfigs[constants.ClientNameAgave].ReleaseNotesRegexes[cluster])
		if err != nil {
			return nil, fmt.Errorf("failed to compile agave release notes regex for jito-solana classification: %w", err)
		}
		versionStrings[cluster] = jitoVersionStringsFromAgaveReleaseBodyRegex(
			jitoReleases,
			agaveReleases,
			agaveReleaseNotesRegex,
			true,
		)
	}

	return c.latestVersionFromClusterVersionStrings(versionStrings)
}

func (c *Client) getLatestRakuraiVersion(ctx context.Context) (latestVersion *version.Version, err error) {
	rakuraiTags, _, err := c.client.Repositories.ListTags(ctx, c.repoOwner, c.repoName, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get rakurai tags: %w", err)
	}

	mainnetTagInfos := tagVersionInfosFromTagRegex(rakuraiTags, c.tagRegexes[constants.ClusterNameMainnetBeta], false)
	testnetTagInfos := tagVersionInfosFromTagRegex(rakuraiTags, c.tagRegexes[constants.ClusterNameTestnet], true)

	c.setCachedTagInfos(append(mainnetTagInfos, testnetTagInfos...))

	selectedTag, err := c.selectRakuraiTagVersionInfo(mainnetTagInfos, testnetTagInfos)
	if err != nil {
		return nil, err
	}

	c.logger.Info("latest version "+selectedTag.Version.Original(),
		"client", c.clientName,
		"cluster", c.cluster,
		"selectedTag", selectedTag.TagName,
		"repoURL", c.repoURL+"/tags",
	)

	return selectedTag.Version, nil
}

func (c *Client) latestVersionFromClusterVersionStrings(versionStrings map[string][]string) (latestVersion *version.Version, err error) {
	// fail if no releases/tags found for client configured cluster
	for cluster, versionStrings := range versionStrings {
		if len(versionStrings) == 0 {
			return nil, fmt.Errorf("no %s versions found for client %s", cluster, c.clientName)
		}
	}

	// For each cluster, create a versions slice and sort, and get the latest version
	latestClusterVersion := make(map[string]*version.Version)
	c.cachedTagVersions = nil
	c.cachedTagInfos = nil
	for cluster, versionStrings := range versionStrings {
		sortedTagInfos := c.sortedTagVersionInfosFromVersionStrings(versionStrings)
		if len(sortedTagInfos) == 0 {
			return nil, fmt.Errorf("no parsable %s versions found for client %s", cluster, c.clientName)
		}
		for i := range sortedTagInfos {
			sortedTagInfos[i].TestnetOnly = cluster == constants.ClusterNameTestnet
		}
		latestClusterVersion[cluster] = sortedTagInfos[len(sortedTagInfos)-1].Version
		for _, tagInfo := range sortedTagInfos {
			c.cachedTagVersions = append(c.cachedTagVersions, tagInfo.Version)
			c.cachedTagInfos = append(c.cachedTagInfos, tagInfo)
		}
		c.logger.Debug("latest version "+latestClusterVersion[cluster].Original(), "client", c.clientName, "cluster", cluster, "repoURL", c.versionSourceURL())
	}

	// If cluster is testnet and mainnet version is higher, use mainnet version and warn
	latestVersion = latestClusterVersion[c.cluster]
	if c.cluster == constants.ClusterNameTestnet && latestClusterVersion[constants.ClusterNameMainnetBeta].GreaterThan(latestVersion) {
		latestVersion = latestClusterVersion[constants.ClusterNameMainnetBeta]
		c.logger.Warn(fmt.Sprintf("mainnet v%s > v%s testnet - preferring mainnet version",
			latestClusterVersion[constants.ClusterNameMainnetBeta].Original(),
			latestClusterVersion[c.cluster].Original()),
			"client", c.clientName, "cluster", c.cluster, "repoURL", c.versionSourceURL())
	}

	c.logger.Info("latest version "+latestVersion.Original(), "client", c.clientName, "cluster", c.cluster, "repoURL", c.versionSourceURL())

	return latestVersion, nil
}

func (c *Client) selectRakuraiTagVersionInfo(mainnetTagInfos []tagVersionInfo, testnetTagInfos []tagVersionInfo) (selected tagVersionInfo, err error) {
	latestMainnet, hasMainnet := latestTagVersionInfo(mainnetTagInfos)
	latestTestnet, hasTestnet := latestTagVersionInfo(testnetTagInfos)

	switch c.cluster {
	case constants.ClusterNameMainnetBeta:
		if !hasMainnet {
			return tagVersionInfo{}, fmt.Errorf("%w for client %s cluster %s", ErrNoMatchingTaggedVersion, c.clientName, c.cluster)
		}
		return latestMainnet, nil

	case constants.ClusterNameTestnet:
		if !hasMainnet && !hasTestnet {
			return tagVersionInfo{}, fmt.Errorf("%w for client %s cluster %s", ErrNoMatchingTaggedVersion, c.clientName, c.cluster)
		}
		if !hasTestnet {
			return latestMainnet, nil
		}
		if !hasMainnet {
			return latestTestnet, nil
		}
		if latestMainnet.Version.GreaterThan(latestTestnet.Version) {
			c.logger.Warn("mainnet/general Rakurai tag is newer than latest testnet-only tag - preferring the higher shared version",
				"mainnetTag", latestMainnet.TagName,
				"testnetTag", latestTestnet.TagName)
			return latestMainnet, nil
		}
		// When equal or greater, prefer the explicit testnet tag.
		return latestTestnet, nil
	}

	return tagVersionInfo{}, fmt.Errorf("unsupported cluster: %s", c.cluster)
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

	if c.clientName == constants.ClientNameRakurai {
		tagInfos := append(
			tagVersionInfosFromTagRegex(tags, c.tagRegexes[constants.ClusterNameMainnetBeta], false),
			tagVersionInfosFromTagRegex(tags, c.tagRegexes[constants.ClusterNameTestnet], true)...,
		)
		for _, tagInfo := range tagInfos {
			c.logger.Debug("comparing rakurai tag version to test version", "tag", tagInfo.TagName, "tagVersion", tagInfo.Version.Core().String(), "testVersion", testVersion.Core().String())
			if tagInfo.Version.Core().Compare(testVersion.Core()) == 0 {
				return true, nil
			}
		}
		return false, nil
	}

	if c.clientName == constants.ClientNameJitoSolana {
		for _, tag := range tags {
			if !jitoVersionSuffixRegex.MatchString(tag.GetName()) {
				continue
			}

			tagInfo, err := c.tagVersionInfoFromVersionString(tag.GetName())
			if err != nil {
				c.logger.Debug("skipping jito-solana tag with unparsable version", "tag", tag.GetName(), "error", err)
				continue
			}

			c.logger.Debug("comparing jito-solana tag version to test version", "tag", tagInfo.TagName, "tagVersion", tagInfo.Version.Original(), "testVersion", testVersion.Original())
			if tagInfo.Version.Equal(testVersion) {
				c.cacheTagInfo(tagInfo)
				return true, nil
			}
		}
		return false, nil
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

func (c *Client) TagNameForVersion(v *version.Version) string {
	if c.clientName == constants.ClientNameRakurai {
		matchingTagInfos := make([]tagVersionInfo, 0)
		for _, tagInfo := range c.cachedTagInfos {
			if tagInfo.Version.Equal(v) || tagInfo.Version.Core().Compare(v.Core()) == 0 {
				matchingTagInfos = append(matchingTagInfos, tagInfo)
			}
		}

		switch c.cluster {
		case constants.ClusterNameTestnet:
			for _, tagInfo := range matchingTagInfos {
				if tagInfo.TestnetOnly {
					return tagInfo.TagName
				}
			}
			for _, tagInfo := range matchingTagInfos {
				if !tagInfo.TestnetOnly {
					return tagInfo.TagName
				}
			}
		default:
			for _, tagInfo := range matchingTagInfos {
				if !tagInfo.TestnetOnly {
					return tagInfo.TagName
				}
			}
		}
	}

	for _, tagInfo := range c.cachedTagInfos {
		if tagInfo.Version.Equal(v) {
			return tagInfo.TagName
		}
	}

	if c.clientName == constants.ClientNameJitoSolana {
		for _, tagInfo := range c.cachedTagInfos {
			if c.jitoTagInfoMatchesVersion(tagInfo, v) {
				return tagInfo.TagName
			}
		}
	}

	return v.Original()
}

// ResolveFiredancerSFDPCompliantVersion maps SFDP Firedancer requirements to
// actual Firedancer repo tags. Legacy Frankendancer tags encode Agave
// compatibility in v0.xxx.yyyyy, while native Firedancer v1+ tags do not.
// SFDP may still publish legacy compatibility-shaped versions like
// 0.101.0-beta.40101, whose repo tag equivalent is v0.1001.40101.
func (c *Client) ResolveFiredancerSFDPCompliantVersion(targetVersion *version.Version, minVersion *version.Version, hasMinVersion bool, maxVersion *version.Version, hasMaxVersion bool) (*version.Version, error) {
	if c.clientName != constants.ClientNameFiredancer {
		return nil, fmt.Errorf("firedancer SFDP resolver called for client %s", c.clientName)
	}

	if isNativeFiredancerVersion(targetVersion) {
		if hasMaxVersion {
			return nil, fmt.Errorf("native firedancer target %s cannot be evaluated against SFDP max requirement %s", targetVersion.Original(), maxVersion.Original())
		}
		return targetVersion, nil
	}

	targetKey, err := firedancerCompatibilityKey(targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse firedancer target compatibility key from %s: %w", targetVersion.Original(), err)
	}

	var minKey int64
	if hasMinVersion {
		minKey, err = firedancerCompatibilityKey(minVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse firedancer SFDP min compatibility key from %s: %w", minVersion.Original(), err)
		}
	}

	var maxKey int64
	if hasMaxVersion {
		maxKey, err = firedancerCompatibilityKey(maxVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse firedancer SFDP max compatibility key from %s: %w", maxVersion.Original(), err)
		}
	}

	if firedancerCompatibilityKeySatisfies(targetKey, minKey, hasMinVersion, maxKey, hasMaxVersion) {
		return targetVersion, nil
	}

	preferHighestCompatible := hasMaxVersion && targetKey > maxKey
	selectedVersion, ok, err := c.selectFiredancerTagByCompatibilityKey(minKey, hasMinVersion, maxKey, hasMaxVersion, preferHighestCompatible)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no firedancer tag in cached release list satisfies SFDP requirement %s", firedancerRequirementString(minVersion, hasMinVersion, maxVersion, hasMaxVersion))
	}

	return selectedVersion, nil
}

func (c *Client) selectFiredancerTagByCompatibilityKey(minKey int64, hasMinVersion bool, maxKey int64, hasMaxVersion bool, preferHighestCompatible bool) (*version.Version, bool, error) {
	candidates := c.cachedTagInfos
	if len(candidates) == 0 {
		candidates = make([]tagVersionInfo, 0, len(c.cachedTagVersions))
		for _, cachedVersion := range c.cachedTagVersions {
			candidates = append(candidates, tagVersionInfo{
				TagName: cachedVersion.Original(),
				Version: cachedVersion,
			})
		}
	}

	var selected tagVersionInfo
	var selectedKey int64
	for _, candidate := range candidates {
		if c.cluster == constants.ClusterNameMainnetBeta && candidate.TestnetOnly {
			continue
		}

		candidateKey, err := firedancerCompatibilityKey(candidate.Version)
		if err != nil {
			c.logger.Debug("skipping firedancer tag with unparsable compatibility key",
				"tag", candidate.TagName,
				"version", candidate.Version.Original(),
				"error", err,
			)
			continue
		}
		if !firedancerCompatibilityKeySatisfies(candidateKey, minKey, hasMinVersion, maxKey, hasMaxVersion) {
			continue
		}

		if selected.Version == nil {
			selected = candidate
			selectedKey = candidateKey
			continue
		}

		if preferHighestCompatible {
			if candidateKey > selectedKey || (candidateKey == selectedKey && candidate.Version.GreaterThan(selected.Version)) {
				selected = candidate
				selectedKey = candidateKey
			}
			continue
		}

		if candidateKey < selectedKey || (candidateKey == selectedKey && candidate.Version.GreaterThan(selected.Version)) {
			selected = candidate
			selectedKey = candidateKey
		}
	}

	if selected.Version == nil {
		return nil, false, nil
	}
	return selected.Version, true, nil
}

func firedancerCompatibilityKeySatisfies(key int64, minKey int64, hasMinVersion bool, maxKey int64, hasMaxVersion bool) bool {
	if hasMinVersion && key < minKey {
		return false
	}
	if hasMaxVersion && key > maxKey {
		return false
	}
	return true
}

func isNativeFiredancerVersion(v *version.Version) bool {
	segments := v.Segments()
	return len(segments) > 0 && segments[0] >= 1
}

func firedancerCompatibilityKey(v *version.Version) (int64, error) {
	if prerelease := v.Prerelease(); prerelease != "" {
		parts := strings.Split(prerelease, ".")
		for i := len(parts) - 1; i >= 0; i-- {
			key, err := strconv.ParseInt(parts[i], 10, 64)
			if err == nil {
				return key, nil
			}
		}
		return 0, fmt.Errorf("pre-release %q has no numeric compatibility component", prerelease)
	}

	segments := v.Segments()
	if len(segments) < 3 {
		return 0, fmt.Errorf("version %q has fewer than three segments", v.Original())
	}
	return int64(segments[2]), nil
}

func firedancerRequirementString(minVersion *version.Version, hasMinVersion bool, maxVersion *version.Version, hasMaxVersion bool) string {
	requirements := make([]string, 0, 2)
	if hasMinVersion {
		requirements = append(requirements, ">= "+minVersion.Original())
	}
	if hasMaxVersion {
		requirements = append(requirements, "<= "+maxVersion.Original())
	}
	if len(requirements) == 0 {
		return "<none>"
	}
	return strings.Join(requirements, ",")
}

func (c *Client) versionSourceURL() string {
	if c.clientName == constants.ClientNameRakurai {
		return c.repoURL + "/tags"
	}
	return c.repoURL + "/releases"
}

func (c *Client) setCachedTagInfos(tagInfos []tagVersionInfo) {
	c.cachedTagInfos = tagInfos
	c.cachedTagVersions = make([]*version.Version, 0, len(tagInfos))
	for _, tagInfo := range tagInfos {
		c.cachedTagVersions = append(c.cachedTagVersions, tagInfo.Version)
	}
}

func (c *Client) cacheTagInfo(tagInfo tagVersionInfo) {
	for _, cached := range c.cachedTagInfos {
		if cached.TagName == tagInfo.TagName {
			return
		}
	}

	c.cachedTagInfos = append(c.cachedTagInfos, tagInfo)
	c.cachedTagVersions = append(c.cachedTagVersions, tagInfo.Version)
}

func (c *Client) jitoTagInfoMatchesVersion(tagInfo tagVersionInfo, v *version.Version) bool {
	if tagInfo.Version.Equal(v) {
		return true
	}

	strippedTagName := jitoVersionSuffixRegex.ReplaceAllString(tagInfo.TagName, "")
	if strippedTagName == tagInfo.TagName {
		return false
	}

	strippedVersion, err := version.NewVersion(strippedTagName)
	if err != nil {
		return false
	}

	return strippedVersion.Equal(v)
}

// NormalizeToTagVersion translates the running version reported by the validator RPC
// into its equivalent git tag version from the cached release list. This is necessary
// because several clients append a client-specific suffix to their git tags that the
// RPC does not include, or encode the version differently from the tag.
//
// Jito-Solana:
//   - RPC reports: 4.0.0-beta.2
//   - Git tag:     v4.0.0-beta.2-jito  (or v3.1.10-jito.1)
//   - Strategy: strip -jito[.N] from each cached tag and compare to the running version
//
// Agave:
//   - RPC reports: 2.2.8-beta.1
//   - Git tag:     v2.2.8-beta.1  (no suffix — already matches)
//   - Strategy: direct equality match against cached tags
//
// Firedancer:
//   - RPC may report: 0.902.0-beta.40002  or  0.33670.40002
//   - Git tag:        v0.902.40002
//   - Strategy 1: match by feature-set (PATCH > 0), e.g. 0.33670.40002 → v0.902.40002
//   - Strategy 2: match by MAJOR.MINOR when PATCH is 0, e.g. 0.902.0 → v0.902.40002
//
// If no match is found the running version is returned unchanged as a safe fallback.
func (c *Client) NormalizeToTagVersion(v *version.Version) *version.Version {
	switch c.clientName {

	case constants.ClientNameJitoSolana:
		// Jito tags carry a -jito[.N] suffix not present in the RPC version.
		// Strip it from each cached tag and compare to find the matching tag version.
		for _, tagged := range c.cachedTagVersions {
			stripped := jitoVersionSuffixRegex.ReplaceAllString(tagged.Original(), "")
			strippedVersion, err := version.NewVersion(stripped)
			if err != nil {
				continue
			}
			if strippedVersion.Equal(v) {
				c.logger.Debug("normalized jito-solana running version to tag version",
					"running", v.Original(), "tag", tagged.Original())
				return tagged
			}
		}
		c.logger.Warn("could not normalize jito-solana running version to tag version - no cached tag matched after stripping -jito suffix",
			"running", v.Original())
		return v

	case constants.ClientNameAgave:
		// Agave tags match the RPC version directly (no client suffix).
		// Perform an explicit lookup so pre-release metadata in the tag Original()
		// is preserved for VersionToTag even when the running version string differs
		// only in formatting (e.g. leading 'v').
		for _, tagged := range c.cachedTagVersions {
			if tagged.Equal(v) {
				return tagged
			}
		}
		// No cached tag found — return unchanged (version already matches or is not yet released)
		return v

	case constants.ClientNameRakurai:
		// Rakurai validator release tags carry a release/ prefix and may optionally
		// include a _testnet suffix in GitHub. Those parts are tracked separately in
		// cachedTagInfos, so here we normalize by matching the semver payload only.
		for _, tagged := range c.cachedTagVersions {
			if tagged.Equal(v) || tagged.Core().Compare(v.Core()) == 0 {
				return tagged
			}
		}
		// No cached tag found — return unchanged (version already matches or is not yet released)
		return v

	case constants.ClientNameFiredancer:
		// If the RPC already reports the tag-shaped version, preserve that exact
		// release before falling back to looser Firedancer matching.
		for _, tagged := range c.cachedTagVersions {
			if tagged.Equal(v) {
				return tagged
			}
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
			var matchingTag *version.Version
			for _, tagged := range c.cachedTagVersions {
				tagSegs := tagged.Segments()
				if len(tagSegs) >= 3 && tagSegs[2] == featureSet {
					if matchingTag == nil || tagged.GreaterThan(matchingTag) {
						matchingTag = tagged
					}
				}
			}
			if matchingTag != nil {
				c.logger.Debug("normalized firedancer running version to tag version (feature-set match)",
					"running", v.Original(), "tag", matchingTag.Original())
				return matchingTag
			}
		}

		// Strategy 2: match by MAJOR.MINOR when PATCH is zero (or feature-set match found nothing).
		// Handles the case where the RPC returns EPOCH.RELEASE.0, e.g. 0.902.0 matching tag v0.902.40002.
		var matchingTag *version.Version
		for _, tagged := range c.cachedTagVersions {
			tagSegs := tagged.Segments()
			if len(tagSegs) >= 3 && tagSegs[0] == segs[0] && tagSegs[1] == segs[1] {
				if matchingTag == nil || tagged.GreaterThan(matchingTag) {
					matchingTag = tagged
				}
			}
		}
		if matchingTag != nil {
			c.logger.Debug("normalized firedancer running version to tag version (major.minor match)",
				"running", v.Original(), "tag", matchingTag.Original())
			return matchingTag
		}

		c.logger.Warn("could not normalize firedancer running version to tag version - no cached tag matched by feature-set or major.minor",
			"running", v.Original(), "featureSet", featureSet)
		return v
	}

	return v
}

// versionsFromReleaseTitleRegex gets versions from non-prerelease releases with titles matching the supplied regex
func versionsFromReleaseTitleRegex(releases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	return versionsFromReleaseTitleRegexWithPrerelease(releases, regex, false)
}

func versionsFromReleaseTitleRegexWithPrerelease(releases []*github.RepositoryRelease, regex *regexp.Regexp, includePrereleases bool) (versionStrings []string) {
	for _, release := range releases {
		if release.GetPrerelease() && !includePrereleases {
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

func versionsFromReleaseTitleAndBodyRegex(releases []*github.RepositoryRelease, titleRegex *regexp.Regexp, bodyRegex *regexp.Regexp) (versionStrings []string) {
	return versionsFromReleaseTitleAndBodyRegexWithPrerelease(releases, titleRegex, bodyRegex, false)
}

func versionsFromReleaseTitleAndBodyRegexWithPrerelease(releases []*github.RepositoryRelease, titleRegex *regexp.Regexp, bodyRegex *regexp.Regexp, includePrereleases bool) (versionStrings []string) {
	for _, release := range releases {
		if release.GetPrerelease() && !includePrereleases {
			log.Debug("skipping github pre-release", "title", release.GetName(), "tag", release.GetTagName())
			continue
		}
		if titleRegex.MatchString(release.GetName()) && bodyRegex.MatchString(release.GetBody()) {
			log.Debug("found matching release by title and notes", "title", release.GetName(), "tag", release.GetTagName(), "version", release.GetTagName())
			versionStrings = append(versionStrings, release.GetTagName())
		}
	}
	return versionStrings
}

func appendUniqueVersionStrings(versionStrings []string, candidates ...string) []string {
	seen := make(map[string]struct{}, len(versionStrings)+len(candidates))
	for _, versionString := range versionStrings {
		seen[versionString] = struct{}{}
	}

	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		versionStrings = append(versionStrings, candidate)
		seen[candidate] = struct{}{}
	}

	return versionStrings
}

func versionsFromTagRegex(tags []*github.RepositoryTag, regex *regexp.Regexp) (versionStrings []string) {
	tagInfos := tagVersionInfosFromTagRegex(tags, regex, false)
	versionStrings = make([]string, 0, len(tagInfos))
	for _, tagInfo := range tagInfos {
		versionStrings = append(versionStrings, tagInfo.Version.Original())
	}
	return versionStrings
}

// versionsFromReleaseBodyRegex gets versions from non-prerelease releases with bodies matching the supplied regex
func versionsFromReleaseBodyRegex(releases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	return versionsFromReleaseBodyRegexWithPrerelease(releases, regex, false)
}

func versionsFromReleaseBodyRegexWithPrerelease(releases []*github.RepositoryRelease, regex *regexp.Regexp, includePrereleases bool) (versionStrings []string) {
	for _, release := range releases {
		if release.GetPrerelease() && !includePrereleases {
			log.Debug("skipping github pre-release", "title", release.GetName(), "tag", release.GetTagName())
			continue
		}
		if regex.MatchString(release.GetBody()) {
			versionStrings = append(versionStrings, release.GetTagName())
		}
	}
	return versionStrings
}

func jitoVersionStringsFromAgaveReleaseBodyRegex(jitoReleases []*github.RepositoryRelease, agaveReleases []*github.RepositoryRelease, regex *regexp.Regexp, includePrereleases bool) (versionStrings []string) {
	agaveVersionKeys := make(map[string]struct{})
	for _, agaveVersionString := range versionsFromReleaseBodyRegexWithPrerelease(agaveReleases, regex, includePrereleases) {
		key, err := versionKey(agaveVersionString)
		if err != nil {
			log.Debug("skipping unparsable agave release version", "version", agaveVersionString, "error", err)
			continue
		}
		agaveVersionKeys[key] = struct{}{}
	}

	for _, release := range jitoReleases {
		if release.GetPrerelease() && !includePrereleases {
			log.Debug("skipping github pre-release", "title", release.GetName(), "tag", release.GetTagName())
			continue
		}

		tagName := release.GetTagName()
		agaveVersionString := jitoVersionSuffixRegex.ReplaceAllString(tagName, "")
		if agaveVersionString == tagName {
			continue
		}

		key, err := versionKey(agaveVersionString)
		if err != nil {
			log.Debug("skipping jito-solana release with unparsable agave version", "title", release.GetName(), "tag", tagName, "version", agaveVersionString, "error", err)
			continue
		}

		if _, ok := agaveVersionKeys[key]; ok {
			log.Debug("found matching jito-solana release by agave classification", "title", release.GetName(), "tag", tagName, "agaveVersion", agaveVersionString)
			versionStrings = append(versionStrings, tagName)
		}
	}

	return versionStrings
}

func versionKey(versionString string) (string, error) {
	v, err := version.NewVersion(versionString)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}

// setOwnerAndRepo extracts owner and repo from a GitHub URL
func (c *Client) setOwnerAndRepo() (err error) {
	c.repoOwner, c.repoName, err = ownerAndRepoFromURL(c.repoURL)
	if err != nil {
		return err
	}

	return nil
}

func ownerAndRepoFromURL(repoURL string) (owner string, repo string, err error) {
	matches := githubRepoAndOwnerFromURLRegex.FindStringSubmatch(repoURL)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("unsupported GitHub URL format: %s", repoURL)
	}

	return matches[1], matches[2], nil
}

func latestTagVersionInfo(tagInfos []tagVersionInfo) (latest tagVersionInfo, ok bool) {
	if len(tagInfos) == 0 {
		return tagVersionInfo{}, false
	}

	latest = tagInfos[0]
	for _, candidate := range tagInfos[1:] {
		if candidate.Version.GreaterThan(latest.Version) {
			latest = candidate
		}
	}
	return latest, true
}

func tagVersionInfosFromTagRegex(tags []*github.RepositoryTag, regex *regexp.Regexp, testnetOnly bool) (tagInfos []tagVersionInfo) {
	for _, tag := range tags {
		matches := regex.FindStringSubmatch(tag.GetName())
		if matches == nil {
			continue
		}

		versionString := tag.GetName()
		if len(matches) > 1 {
			versionString = matches[1]
		}

		parsedVersion, err := version.NewVersion(versionString)
		if err != nil {
			log.Debug("skipping tag with unparsable version", "tag", tag.GetName(), "versionString", versionString, "error", err)
			continue
		}

		log.Debug("found matching tag", "tag", tag.GetName(), "version", parsedVersion.Original(), "testnetOnly", testnetOnly)
		tagInfos = append(tagInfos, tagVersionInfo{
			TagName:     tag.GetName(),
			Version:     parsedVersion,
			TestnetOnly: testnetOnly,
		})
	}
	return tagInfos
}

func (c *Client) sortedTagVersionInfosFromVersionStrings(versionStrings []string) (sortedTagInfos []tagVersionInfo) {
	c.logger.Debug("sorting versions", "versionStrings", versionStrings)
	sortedTagInfos = make([]tagVersionInfo, 0, len(versionStrings))
	for _, raw := range versionStrings {
		tagInfo, err := c.tagVersionInfoFromVersionString(raw)
		if err != nil {
			c.logger.Debug("skipping unparsable version", "version", raw, "error", err)
			continue
		}
		sortedTagInfos = append(sortedTagInfos, tagInfo)
	}
	sort.Slice(sortedTagInfos, func(i, j int) bool {
		if !sortedTagInfos[i].Version.Equal(sortedTagInfos[j].Version) {
			return sortedTagInfos[i].Version.LessThan(sortedTagInfos[j].Version)
		}
		return versionTagLess(sortedTagInfos[i].TagName, sortedTagInfos[j].TagName)
	})
	c.logger.Debug("sorted versions", "sortedVersions", sortedTagInfos)
	return sortedTagInfos
}

func (c *Client) tagVersionInfoFromVersionString(raw string) (tagVersionInfo, error) {
	versionString := raw
	if c.clientName == constants.ClientNameJitoSolana {
		// Jito tags append -jito[.N] to the upstream Agave version. Compare on
		// the Agave version so stable releases sort above their release candidates.
		versionString = jitoVersionSuffixRegex.ReplaceAllString(raw, "")
	}

	parsedVersion, err := version.NewVersion(versionString)
	if err != nil {
		return tagVersionInfo{}, err
	}

	return tagVersionInfo{
		TagName: raw,
		Version: parsedVersion,
	}, nil
}

func versionTagLess(a, b string) bool {
	parsedA, errA := version.NewVersion(a)
	parsedB, errB := version.NewVersion(b)
	if errA == nil && errB == nil && !parsedA.Equal(parsedB) {
		return parsedA.LessThan(parsedB)
	}
	return a < b
}
