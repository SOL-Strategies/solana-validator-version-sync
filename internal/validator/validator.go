package validator

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-version"
	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
	"github.com/sol-strategies/solana-validator-version-sync/internal/github"
	"github.com/sol-strategies/solana-validator-version-sync/internal/rpc"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sfdp"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sync_commands"
	"github.com/sol-strategies/solana-validator-version-sync/internal/versiondiff"
)

const (
	// RoleActive is the role of the validator that is active
	RoleActive = "active"
	// RolePassive is the role of the validator that is passive
	RolePassive = "passive"
	// RoleUnknown is the role of the validator that is unknown
	RoleUnknown = "unknown"
)

// Options represents the options for creating a new Validator
type Options struct {
	Cluster         string
	SyncConfig      config.Sync
	ValidatorConfig config.Validator
}

// Validator represents the validator - its state can be refreshed with the RefreshState method
type Validator struct {
	ActiveIdentityPublicKey  string
	PassiveIdentityPublicKey string
	State                    State

	syncConfig   config.Sync
	cfg          config.Validator
	logger       *log.Logger
	rpcClient    *rpc.Client
	sfdpClient   *sfdp.Client
	githubClient *github.Client
}

// New creates a new Validator
func New(opts Options) (v *Validator, err error) {
	v = &Validator{
		State: State{
			Cluster: opts.Cluster,
		},
		ActiveIdentityPublicKey:  opts.ValidatorConfig.Identities.ActiveKeyPair.PublicKey().String(),
		PassiveIdentityPublicKey: opts.ValidatorConfig.Identities.PassiveKeyPair.PublicKey().String(),
		syncConfig:               opts.SyncConfig,
		cfg:                      opts.ValidatorConfig,
		logger:                   log.WithPrefix("validator"),
	}

	// Create clients
	v.rpcClient = rpc.NewClient(v.cfg.RPCURL)
	v.githubClient, err = github.NewClient(github.Options{
		Cluster: opts.Cluster,
		Client:  v.cfg.Client,
	})
	v.sfdpClient = sfdp.NewClient(sfdp.Options{
		Cluster: opts.Cluster,
		Client:  v.cfg.Client,
	})

	// Parse commands after copying the config
	for i := range v.syncConfig.Commands {
		err = v.syncConfig.Commands[i].Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse command %d (%s): %w", i, v.syncConfig.Commands[i].Name, err)
		}
	}

	return v, nil
}

// SyncVersion syncs the validator's version
func (v *Validator) SyncVersion() (err error) {
	// warn if active and passive identites are the same
	if v.ActiveIdentityPublicKey == v.PassiveIdentityPublicKey {
		v.logger.Warn("configured active and passive identites are the same",
			"activePubkey", v.ActiveIdentityPublicKey,
			"passivePubkey", v.PassiveIdentityPublicKey,
		)
	}

	// refresh the validator's state
	err = v.refreshState()
	if err != nil {
		return err
	}

	syncLogger := log.WithPrefix("sync").With(
		"client", v.cfg.Client,
		"version", v.State.Version.String(),
		"role", v.Role(),
	)

	// decide if we should sync based on the validator's role and the enabled when active config
	switch v.Role() {
	case RoleActive:
		if !v.syncConfig.EnabledWhenActive {
			syncLogger.Warnf("validator is %s and we don't run with scissors ❌🏃✂️  - skipping sync (allow with sync.enabled_when_active=true)", v.Role())
			return nil
		}
		syncLogger.Warnf("validator is %s and sync.enabled_when_active=%t running with scissors ⚠️🏃‍♂️✂️  - syncing", v.Role(), v.syncConfig.EnabledWhenActive)
	case RolePassive:
		syncLogger.Info("validator is passive - syncing")
	default:
		return fmt.Errorf("validator identity public key %s is not %s or %s - skipping sync", v.State.IdentityPublicKey, RoleActive, RolePassive)
	}

	// set a version we'll target as part of a diff
	syncLogger.Debug("creating version diff", "from", v.State.Version, "fromString", v.State.VersionString)
	versionDiff := versiondiff.VersionDiff{
		From: v.State.Version,
	}

	// by default target the latest client version for the cluster
	versionDiff.To, err = v.githubClient.GetLatestClientVersion()
	if err != nil {
		return err
	}

	syncLogger.Debug("latest release from repo", "version", versionDiff.To.String())

	// If enabled, ensure target version is within SFDP constraints or update to max/min allowed SFDP version
	if v.syncConfig.EnableSFDPCompliance {
		sfdpRequirements, err := v.sfdpClient.GetLatestRequirements()
		if err != nil {
			return err
		}

		syncLogger.Debug("got latest requirements from SFDP", "sfdpRequirements", sfdpRequirements.ConstraintsString)

		// if target version is not within sfdp constraints, update it to get the max version sfdp allows
		if sfdpRequirements.Constraints.Check(versionDiff.To.Core()) {
			syncLogger.Info("target version is within SFDP constraints",
				"targetVersion", versionDiff.To.Core().String(),
				"sfdpRequirement", sfdpRequirements.ConstraintsString,
			)
		} else if sfdpRequirements.HasMaxVersion {
			syncLogger.Warn("target version is not within SFDP constraints - updating to max allowed SFDP version",
				"targetVersion", versionDiff.To.Core().String(),
				"sfdpMaxVersion", sfdpRequirements.MaxVersion.String(),
				"sfdpRequirement", sfdpRequirements.ConstraintsString,
			)
			versionDiff.To = sfdpRequirements.MaxVersion
		} else if sfdpRequirements.HasMinVersion {
			syncLogger.Warn("target version is not within SFDP constraints - updating to min allowed SFDP version",
				"targetVersion", versionDiff.To.Core().String(),
				"sfdpMinVersion", sfdpRequirements.MinVersion.String(),
				"sfdpRequirement", sfdpRequirements.ConstraintsString,
			)
			versionDiff.To = sfdpRequirements.MinVersion
		}
	}

	syncLogger.Debugf("final target sync version: %s", versionDiff.To.Core().String())
	syncLogger = syncLogger.With("targetVersion", versionDiff.To.Core().String())

	// if already on the target version, do nothing
	if versionDiff.IsSameVersion() {
		syncLogger.Info("validator already running target version - nothing to do")
		return nil
	}

	// if not allowed to sync major, minor, or patch, - warn and do nothing
	if !v.syncConfig.AllowedSemverChanges.Major && !v.syncConfig.AllowedSemverChanges.Minor && !v.syncConfig.AllowedSemverChanges.Patch {
		syncLogger.Warn("sync.allowed_semver_changes config settings do not allow any version changes - not syncing")
		return nil
	}

	// target major version but not allowed - warn and do nothing
	if versionDiff.HasMajorChange() && !v.syncConfig.AllowedSemverChanges.Major {
		syncLogger.Warn("target version contains major semver change and not allowed from config- not syncing")
		return nil
	}

	// target minor version but not allowed - warn and do nothing
	if versionDiff.HasMinorChange() && !v.syncConfig.AllowedSemverChanges.Minor {
		syncLogger.Warn("target version contains minor semver change and not allowed from config- not syncing")
		return nil
	}

	// target patch version but not allowed - warn and do nothing
	if versionDiff.HasPatchChange() && !v.syncConfig.AllowedSemverChanges.Patch {
		syncLogger.Warn("target version contains patch semver change and not allowed from config- not syncing")
		return nil
	}

	// by now we know we need to sync and are allowed to sync to the target version
	syncLogger = syncLogger.With("syncDirection", versionDiff.Direction())
	syncLogger.Infof("%v  %s required v%s -> v%s",
		versionDiff.DirectionEmoji(), versionDiff.Direction(),
		versionDiff.From.Core().String(), versionDiff.To.Core().String(),
	)

	// create the commands
	syncLogger.Infof("executing %d commands", len(v.syncConfig.Commands))
	for cmd_i, cmd := range v.syncConfig.Commands {
		err := cmd.ExecuteWithData(sync_commands.CommandTemplateData{
			CommandIndex:                cmd_i,
			ValidatorClient:             v.cfg.Client,
			ValidatorRPCURL:             v.cfg.RPCURL,
			ValidatorRole:               v.Role(),
			ValidatorRoleIsPassive:      v.IsPassive(),
			ValidatorRoleIsActive:       v.IsActive(),
			ValidatorIdentityPublicKey:  v.State.IdentityPublicKey,
			ClusterName:                 v.State.Cluster,
			Hostname:                    v.State.Hostname,
			VersionFrom:                 versionDiff.From.Core().String(),
			VersionTo:                   versionDiff.To.Core().String(),
			SyncIsSFDPComplianceEnabled: v.syncConfig.EnableSFDPCompliance,
		})
		if err != nil {
			return err
		}
	}

	syncLogger.Info("commands executed successfully")
	return nil
}

// refreshState refreshes the validator's state
func (v *Validator) refreshState() error {
	v.logger.Debug("refreshing validator state")

	// get the validator's hostname
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	v.State.Hostname = hostname

	// get the validator's version string
	versionString, err := v.rpcClient.GetVersion()
	if err != nil {
		return err
	}
	v.State.VersionString = versionString

	// parse the version string
	v.State.Version, err = version.NewVersion(v.State.VersionString)
	if err != nil {
		return err
	}

	// get the validator's identity public key
	identityPubkey, err := v.rpcClient.GetIdentity()
	if err != nil {
		return err
	}
	v.State.IdentityPublicKey = identityPubkey

	// get the validator's health
	health, err := v.rpcClient.GetHealth()
	if err != nil {
		return err
	}
	v.State.HealthStatus = health

	// warn if the validator is running with an identity that does not match active or passive identities
	if v.IsRoleUnknown() {
		v.logger.Warn("validator is running with an identity that does not match active or passive identities",
			"identityPubkey", v.State.IdentityPublicKey,
			"activePubkey", v.ActiveIdentityPublicKey,
			"passivePubkey", v.PassiveIdentityPublicKey,
		)
	}

	v.logger.Debug("validator state refreshed")

	return nil
}

// Role gets the role of the validator
func (v *Validator) Role() string {
	if v.IsActive() {
		return RoleActive
	}
	if v.IsPassive() {
		return RolePassive
	}
	return RoleUnknown
}

// IsRoleUnknown checks if the validator is running with an identity that does not match active or passive identities
func (v *Validator) IsRoleUnknown() bool {
	return v.Role() == RoleUnknown
}

// IsActive checks if the validator is the active identity
func (v *Validator) IsActive() bool {
	return v.State.IdentityPublicKey == v.ActiveIdentityPublicKey
}

// IsPassive checks if the validator is the passive identity
// cover cases like testnet where a validator could be given the same active and passive identity
// in that case, we assume active
func (v *Validator) IsPassive() bool {
	return v.State.IdentityPublicKey == v.PassiveIdentityPublicKey && !v.IsActive()
}
