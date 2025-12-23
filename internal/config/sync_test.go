package config

import (
	"testing"

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
