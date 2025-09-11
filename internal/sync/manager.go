package sync

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
	"github.com/sol-strategies/solana-validator-version-sync/internal/github"
	"github.com/sol-strategies/solana-validator-version-sync/internal/rpc"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sfdp"
)

// Manager manages the version sync process
type Manager struct {
	cfg    *config.Config
	logger *log.Logger
	rpc    *rpc.Client
	sfdp   *sfdp.Client
	github *github.Client
}

// NewManagerOptions represents options for creating a new Manager
type NewManagerOptions struct {
	Cfg *config.Config
}

// NewManager creates a new Manager
func NewManager(opts NewManagerOptions) *Manager {
	return &Manager{
		cfg:    opts.Cfg,
		logger: log.WithPrefix("sync"),
		rpc:    rpc.NewClient(opts.Cfg.Validator.RPCURL),
		sfdp:   sfdp.NewClient(),
		github: github.NewClient(),
	}
}

// Run starts the sync manager and runs the sync loop
func (m *Manager) Run() error {
	m.logger.Infof("🚀 starting solana-validator-version-sync every %s", m.cfg.Sync.ParsedIntervalDuration)

	// Start the sync loop
	ticker := time.NewTicker(m.cfg.Sync.ParsedIntervalDuration)
	defer ticker.Stop()

	// Run initial sync
	m.runSyncCheck()

	// Run sync loop
	for {
		select {
		case <-ticker.C:
			m.runSyncCheck()
		}
	}
}

func (m *Manager) runSyncCheck() {
	m.logger.Info("================================================")
	m.logger.Info("starting sync check")
	m.runSync()
	m.logger.Infof("next sync check in %s at %s",
		m.cfg.Sync.ParsedIntervalDuration, time.Now().UTC().Add(m.cfg.Sync.ParsedIntervalDuration).Format("2006-01-02T15:04:05Z"),
	)
	m.logger.Info("================================================")
}

// runSync performs a single sync iteration
func (m *Manager) runSync() {
	// Get current validator state
	validatorState, err := m.getValidatorState()
	if err != nil {
		m.logger.Error("failed to get validator state", "error", err)
		return
	}

	m.logger.Info("validator state",
		"cluster", m.cfg.Cluster.Name,
		"client", m.cfg.Validator.Client,
		"running_version", validatorState.RunningVersion,
		"identity_pubkey", validatorState.IdentityPubkey)

	// Check if validator is in SFDP
	sfdpValidator, err := m.sfdp.GetValidator(validatorState.IdentityPubkey)
	if err != nil {
		m.logger.Warn("failed to get validator from SFDP", "error", err)
	} else {
		m.logger.Info("validator found in SFDP", "state", sfdpValidator.State)
	}

	// Get SFDP requirements if compliance is enabled
	var sfdpRequirements *sfdp.Requirements
	if m.cfg.Sync.EnableSFDPCompliance {
		sfdpRequirements, err = m.sfdp.GetRequirements(m.cfg.Cluster.Name)
		if err != nil {
			m.logger.Warn("failed to get SFDP requirements", "error", err)
		} else {
			m.logger.Info("SFDP requirements",
				"min_version", sfdpRequirements.MinVersion,
				"max_version", sfdpRequirements.MaxVersion)
		}
	}

	// Get available versions from GitHub
	availableVersions, err := m.getAvailableVersions()
	if err != nil {
		m.logger.Error("failed to get available versions", "error", err)
		return
	}

	m.logger.Info("available versions", "versions", availableVersions)

	// Determine if sync is needed
	syncAction := m.determineSyncAction(validatorState.RunningVersion, availableVersions, sfdpRequirements)

	switch syncAction.Action {
	case "upgrade":
		m.logger.Info("👍 upgrade needed", "from", validatorState.RunningVersion, "to", syncAction.TargetVersion)
		if err := m.executeSync(syncAction, validatorState); err != nil {
			m.logger.Error("failed to execute sync", "error", err)
			return
		}
	case "downgrade":
		m.logger.Info("👎 downgrade needed", "from", validatorState.RunningVersion, "to", syncAction.TargetVersion)
		if err := m.executeSync(syncAction, validatorState); err != nil {
			m.logger.Error("failed to execute sync", "error", err)
			return
		}
	case "nochange":
		m.logger.Info("👌 no sync needed", "version", validatorState.RunningVersion)
		return
	default:
		m.logger.Error("unknown sync action", "action", syncAction.Action)
		return
	}
}

// SyncAction represents a sync action to be performed
type SyncAction struct {
	Action        string // "upgrade", "downgrade", "nochange"
	TargetVersion string
	Reason        string
}

// getValidatorState gets the current state of the validator
func (m *Manager) getValidatorState() (*rpc.ValidatorState, error) {
	return m.rpc.GetValidatorState()
}

// getAvailableVersions gets available versions for the configured client and cluster
func (m *Manager) getAvailableVersions() ([]string, error) {
	client := m.cfg.Validator.Client
	cluster := m.cfg.Cluster.Name

	// Get repository configuration
	repo, exists := m.cfg.Sync.ClientSourceRepositories[client]
	if !exists {
		return nil, fmt.Errorf("no repository configuration found for client: %s", client)
	}

	// Get release notes regex for the cluster
	regex, exists := repo.ReleaseNotesRegexes[cluster]
	if !exists {
		return nil, fmt.Errorf("no release notes regex found for client %s and cluster %s", client, cluster)
	}

	// Get available versions from GitHub
	return m.github.GetAvailableVersions(repo.URL, regex)
}

// determineSyncAction determines what sync action should be performed
func (m *Manager) determineSyncAction(currentVersion string, availableVersions []string, sfdpRequirements *sfdp.Requirements) SyncAction {
	// TODO: Implement version comparison logic
	// For now, return nochange
	return SyncAction{
		Action:        "nochange",
		TargetVersion: currentVersion,
		Reason:        "version comparison logic not yet implemented",
	}
}

// executeSync executes the sync commands
func (m *Manager) executeSync(action SyncAction, validatorState *rpc.ValidatorState) error {
	m.logger.Info("executing sync", "action", action.Action, "target_version", action.TargetVersion)

	// TODO: Implement command execution logic
	// For now, just log what would be done
	for i, cmd := range m.cfg.Sync.Commands {
		if cmd.Disabled {
			m.logger.Info("skipping disabled command", "index", i, "name", cmd.Name)
			continue
		}

		if cmd.DryRun {
			m.logger.Info("dry run command", "index", i, "name", cmd.Name, "cmd", cmd.Cmd, "args", cmd.Args)
		} else {
			m.logger.Info("would execute command", "index", i, "name", cmd.Name, "cmd", cmd.Cmd, "args", cmd.Args)
		}
	}

	return nil
}
