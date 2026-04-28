package github

import (
	"context"
	"errors"
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
			versionStrings[cluster] = versionsFromReleaseBodyRegex(releases, c.releaseNotesRegexes[cluster])
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
		versionStrings := make(map[string][]string)
		// firedancer flags release cluster in release title prefix
		for _, cluster := range constants.ValidClusterNames {
			versionStrings[cluster] = versionsFromReleaseTitleRegex(releases, c.releaseTitleRegexes[cluster])
		}
		return c.latestVersionFromClusterVersionStrings(versionStrings)
	case constants.ClientNameRakurai:
		return c.getLatestRakuraiVersion(ctx)
	default:
		return nil, fmt.Errorf("unsupported client: %s", c.clientName)
	}
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
		sortedVersions := c.sortedVersionsFromVersionStrings(versionStrings)
		latestClusterVersion[cluster] = sortedVersions[len(sortedVersions)-1]
		c.cachedTagVersions = append(c.cachedTagVersions, sortedVersions...)
		for _, tagged := range sortedVersions {
			c.cachedTagInfos = append(c.cachedTagInfos, tagVersionInfo{
				TagName: tagged.Original(),
				Version: tagged,
			})
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

	return v.Original()
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

func jitoVersionStringsFromAgaveReleaseBodyRegex(jitoReleases []*github.RepositoryRelease, agaveReleases []*github.RepositoryRelease, regex *regexp.Regexp) (versionStrings []string) {
	agaveVersionKeys := make(map[string]struct{})
	for _, agaveVersionString := range versionsFromReleaseBodyRegex(agaveReleases, regex) {
		key, err := versionKey(agaveVersionString)
		if err != nil {
			log.Debug("skipping unparsable agave release version", "version", agaveVersionString, "error", err)
			continue
		}
		agaveVersionKeys[key] = struct{}{}
	}

	for _, release := range jitoReleases {
		if release.GetPrerelease() {
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
