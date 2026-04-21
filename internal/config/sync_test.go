package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sync_commands"
)

func TestSync_Validate(t *testing.T) {
	tests := []struct {
		name    string
		sync    Sync
		wantErr bool
	}{
		{
			name: "valid sync configuration",
			sync: Sync{
				EnabledWhenActive:    true,
				EnableSFDPCompliance: false,
				Commands:             []sync_commands.Command{},
			},
			wantErr: false,
		},
		{
			name: "sync disabled",
			sync: Sync{
				EnabledWhenActive:    false,
				EnableSFDPCompliance: true,
				Commands:             []sync_commands.Command{},
			},
			wantErr: false,
		},
		{
			name: "sync with SFDP compliance enabled",
			sync: Sync{
				EnabledWhenActive:    true,
				EnableSFDPCompliance: true,
				Commands:             []sync_commands.Command{},
			},
			wantErr: false,
		},
		{
			name: "sync with enabled_when_no_active_leader_in_gossip",
			sync: Sync{
				EnabledWhenActive:                 false,
				EnabledWhenNoActiveLeaderInGossip: true,
				EnableSFDPCompliance:              false,
				Commands:                          []sync_commands.Command{},
			},
			wantErr: false,
		},
		{
			name: "sync with both enabled flags",
			sync: Sync{
				EnabledWhenActive:                 true,
				EnabledWhenNoActiveLeaderInGossip: true,
				EnableSFDPCompliance:              false,
				Commands:                          []sync_commands.Command{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sync.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Sync.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSync_Validate_WarnsWhenEnvironmentConfiguredWithoutInheritance(t *testing.T) {
	var output bytes.Buffer

	originalLogger := syncValidationLogger
	syncValidationLogger = log.New(&output).WithPrefix("config")
	t.Cleanup(func() {
		syncValidationLogger = originalLogger
	})

	sync := Sync{
		Commands: []sync_commands.Command{
			{
				Name: "build",
				Environment: map[string]string{
					"TO_VERSION": "1.2.3",
				},
				InheritEnvironment: false,
			},
		},
	}

	if err := sync.Validate(); err != nil {
		t.Fatalf("Sync.Validate() error = %v, want nil", err)
	}

	logged := output.String()
	if !strings.Contains(logged, "inherit_environment=false") {
		t.Fatalf("Sync.Validate() warning missing inherit_environment context: %q", logged)
	}
	if !strings.Contains(logged, "build") {
		t.Fatalf("Sync.Validate() warning missing command name: %q", logged)
	}
}

func TestSync_Validate_DoesNotWarnWhenEnvironmentInheritanceEnabled(t *testing.T) {
	var output bytes.Buffer

	originalLogger := syncValidationLogger
	syncValidationLogger = log.New(&output).WithPrefix("config")
	t.Cleanup(func() {
		syncValidationLogger = originalLogger
	})

	sync := Sync{
		Commands: []sync_commands.Command{
			{
				Name: "build",
				Environment: map[string]string{
					"TO_VERSION": "1.2.3",
				},
				InheritEnvironment: true,
			},
		},
	}

	if err := sync.Validate(); err != nil {
		t.Fatalf("Sync.Validate() error = %v, want nil", err)
	}

	if logged := output.String(); logged != "" {
		t.Fatalf("Sync.Validate() logged warning unexpectedly: %q", logged)
	}
}

func TestSync_SetDefaults(t *testing.T) {
	sync := Sync{}
	sync.SetDefaults()

	// Since SetDefaults is currently empty, we just verify it doesn't panic
	// and the struct remains in its initial state
	if sync.EnabledWhenActive != false {
		t.Errorf("Expected EnabledWhenActive to be false, got %v", sync.EnabledWhenActive)
	}
	if sync.EnabledWhenNoActiveLeaderInGossip != false {
		t.Errorf("Expected EnabledWhenNoActiveLeaderInGossip to be false, got %v", sync.EnabledWhenNoActiveLeaderInGossip)
	}
	if sync.EnableSFDPCompliance != false {
		t.Errorf("Expected EnableSFDPCompliance to be false, got %v", sync.EnableSFDPCompliance)
	}
}

func TestSync_StructFields(t *testing.T) {
	commands := []sync_commands.Command{}
	sync := Sync{
		EnabledWhenActive:                 true,
		EnabledWhenNoActiveLeaderInGossip: true,
		EnableSFDPCompliance:              false,
		Commands:                          commands,
	}

	if sync.EnabledWhenActive != true {
		t.Errorf("Expected EnabledWhenActive to be true, got %v", sync.EnabledWhenActive)
	}
	if sync.EnabledWhenNoActiveLeaderInGossip != true {
		t.Errorf("Expected EnabledWhenNoActiveLeaderInGossip to be true, got %v", sync.EnabledWhenNoActiveLeaderInGossip)
	}
	if sync.EnableSFDPCompliance != false {
		t.Errorf("Expected EnableSFDPCompliance to be false, got %v", sync.EnableSFDPCompliance)
	}
	if len(sync.Commands) != 0 {
		t.Errorf("Expected Commands to be empty, got %v", len(sync.Commands))
	}
}
