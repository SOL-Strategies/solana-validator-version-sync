package config

import (
	"fmt"
	"time"
)

// Sync represents the version sync configuration
type Sync struct {
	// IntervalDuration is the interval at which to run the sync process
	IntervalDuration string `koanf:"interval_duration"`
	// ParsedIntervalDuration is the parsed interval duration
	ParsedIntervalDuration time.Duration `koanf:"-"`
	// ClientSourceRepositories is the configuration for client source repositories
	ClientSourceRepositories map[string]ClientSourceRepository `koanf:"client_source_repositories"`
	// AllowedSemverChanges defines allowed semver changes for the given client
	AllowedSemverChanges AllowedSemverChanges `koanf:"allowed_semver_changes"`
	// EnableSFDPCompliance enables SFDP compliance checking
	EnableSFDPCompliance bool `koanf:"enable_sfdp_compliance"`
	// Commands are the commands to run when there is a version change
	Commands []Command `koanf:"commands"`
}

// ClientSourceRepository represents configuration for a client source repository
type ClientSourceRepository struct {
	// URL is the git repository URL
	URL string `koanf:"url"`
	// ReleaseNotesRegexes are regexes to match release notes for the given cluster
	ReleaseNotesRegexes map[string]string `koanf:"release_notes_regexes"`
}

// AllowedSemverChanges defines allowed semver changes
type AllowedSemverChanges struct {
	// Major allows syncing on major version changes
	Major bool `koanf:"major"`
	// Minor allows syncing on minor version changes
	Minor bool `koanf:"minor"`
	// Patch allows syncing on patch version changes
	Patch bool `koanf:"patch"`
}

// Command represents a command to run during version sync
type Command struct {
	// Name is the name of the command for logging purposes
	Name string `koanf:"name"`
	// Disabled if true, the command will be ignored
	Disabled bool `koanf:"disabled"`
	// DryRun if true, the command will not be executed, only logged
	DryRun bool `koanf:"dry_run"`
	// AllowFailure if true, a non-zero exit code will be logged but subsequent commands will run
	AllowFailure bool `koanf:"allow_failure"`
	// MustSucceed if true, a non-zero exit code will prevent subsequent commands from running
	MustSucceed bool `koanf:"must_succeed"`
	// Cmd is the command to run
	Cmd string `koanf:"cmd"`
	// Args are the arguments to pass to the command
	Args []string `koanf:"args"`
}

// SetDefaults sets default values for the sync configuration
func (s *Sync) SetDefaults() {
	if s.IntervalDuration == "" {
		s.IntervalDuration = "10m"
	}
	if s.AllowedSemverChanges.Minor == false && s.AllowedSemverChanges.Patch == false {
		s.AllowedSemverChanges.Minor = true
		s.AllowedSemverChanges.Patch = true
	}
}

// Validate validates the sync configuration
func (s *Sync) Validate() error {
	// Parse interval duration
	duration, err := time.ParseDuration(s.IntervalDuration)
	if err != nil {
		return fmt.Errorf("sync.interval_duration must be a valid duration (e.g., 10m, 1h), got: %s", s.IntervalDuration)
	}
	s.ParsedIntervalDuration = duration

	// Validate client source repositories
	for client, repo := range s.ClientSourceRepositories {
		if repo.URL == "" {
			return fmt.Errorf("sync.client_source_repositories.%s.url is required", client)
		}
		if len(repo.ReleaseNotesRegexes) == 0 {
			return fmt.Errorf("sync.client_source_repositories.%s.release_notes_regexes is required", client)
		}
	}

	// Validate commands
	for i, cmd := range s.Commands {
		if cmd.Name == "" {
			return fmt.Errorf("sync.commands[%d].name is required", i)
		}
		if cmd.Cmd == "" {
			return fmt.Errorf("sync.commands[%d].cmd is required", i)
		}
	}

	return nil
}
