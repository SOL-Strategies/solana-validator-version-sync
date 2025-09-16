package manager

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
	"github.com/sol-strategies/solana-validator-version-sync/internal/validator"
)

// Manager manages the validator version sync process
type Manager struct {
	cfg       *config.Config
	logger    *log.Logger
	validator *validator.Validator
}

// NewFromConfig creates a new Manager from an already loaded config
func NewFromConfig(cfg *config.Config) (m *Manager, err error) {
	m = &Manager{
		cfg:    cfg,
		logger: log.WithPrefix("manager"),
	}

	// Create validator
	m.validator, err = validator.New(validator.Options{
		Cluster:         cfg.Cluster.Name,
		ValidatorConfig: cfg.Validator,
		SyncConfig:      cfg.Sync,
	})

	if err != nil {
		return nil, err
	}

	// manager created
	m.logger.Debug("created manager from config", "config", cfg)
	return m, nil
}

// RunOnce runs a single sync check and exits
func (m *Manager) RunOnce() error {
	m.logger.Info("🚀 starting solana-validator-version-sync (single run mode)")
	return m.validator.SyncVersion()
}

// RunOnInterval runs the sync manager continuously at the specified interval, errors are logged but not returned after parsing the interval duration string
func (m *Manager) RunOnInterval(intervalDuration time.Duration) (err error) {
	m.logger.Info("🚀 starting solana-validator-version-sync (continuous mode)", "interval", intervalDuration.String())

	// Run sync on a loop with sleep between syncs
	for {
		m.runSyncVersionInterval(intervalDuration)
		time.Sleep(intervalDuration)
	}
}

// runSyncVersionInterval runs the sync version and logs the result without returning an error - used with on interval mode
func (m *Manager) runSyncVersionInterval(intervalDuration time.Duration) {
	m.logger.Info("running sync")
	err := m.validator.SyncVersion()
	nextSyncTime := time.Now().UTC().Add(intervalDuration)

	// Set result string
	resultString := "succeeded"
	if err != nil {
		resultString = "failed"
	}

	msg := fmt.Sprintf("sync %s - next sync in %s at %s",
		resultString, intervalDuration.String(), nextSyncTime.Format("2006-01-02T15:04:05Z"),
	)

	if err != nil {
		m.logger.Error(msg)
	} else {
		m.logger.Info(msg)
	}
}
