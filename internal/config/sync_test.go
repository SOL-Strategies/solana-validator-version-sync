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
				AllowedSemverChanges: AllowedSemverChanges{
					Major: true,
					Minor: true,
					Patch: true,
				},
				Commands: []sync_commands.Command{},
			},
			wantErr: false,
		},
		{
			name: "sync disabled",
			sync: Sync{
				EnabledWhenActive:    false,
				EnableSFDPCompliance: true,
				AllowedSemverChanges: AllowedSemverChanges{
					Major: false,
					Minor: false,
					Patch: false,
				},
				Commands: []sync_commands.Command{},
			},
			wantErr: false,
		},
		{
			name: "only patch changes allowed",
			sync: Sync{
				EnabledWhenActive:    true,
				EnableSFDPCompliance: false,
				AllowedSemverChanges: AllowedSemverChanges{
					Major: false,
					Minor: false,
					Patch: true,
				},
				Commands: []sync_commands.Command{},
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
	if sync.EnableSFDPCompliance != false {
		t.Errorf("Expected EnableSFDPCompliance to be false, got %v", sync.EnableSFDPCompliance)
	}
}

func TestAllowedSemverChanges_StructFields(t *testing.T) {
	changes := AllowedSemverChanges{
		Major: true,
		Minor: false,
		Patch: true,
	}

	if changes.Major != true {
		t.Errorf("Expected Major to be true, got %v", changes.Major)
	}
	if changes.Minor != false {
		t.Errorf("Expected Minor to be false, got %v", changes.Minor)
	}
	if changes.Patch != true {
		t.Errorf("Expected Patch to be true, got %v", changes.Patch)
	}
}

func TestSync_StructFields(t *testing.T) {
	commands := []sync_commands.Command{}
	sync := Sync{
		EnabledWhenActive:    true,
		EnableSFDPCompliance: false,
		AllowedSemverChanges: AllowedSemverChanges{
			Major: true,
			Minor: true,
			Patch: false,
		},
		Commands: commands,
	}

	if sync.EnabledWhenActive != true {
		t.Errorf("Expected EnabledWhenActive to be true, got %v", sync.EnabledWhenActive)
	}
	if sync.EnableSFDPCompliance != false {
		t.Errorf("Expected EnableSFDPCompliance to be false, got %v", sync.EnableSFDPCompliance)
	}
	if sync.AllowedSemverChanges.Major != true {
		t.Errorf("Expected Major to be true, got %v", sync.AllowedSemverChanges.Major)
	}
	if sync.AllowedSemverChanges.Minor != true {
		t.Errorf("Expected Minor to be true, got %v", sync.AllowedSemverChanges.Minor)
	}
	if sync.AllowedSemverChanges.Patch != false {
		t.Errorf("Expected Patch to be false, got %v", sync.AllowedSemverChanges.Patch)
	}
	if len(sync.Commands) != 0 {
		t.Errorf("Expected Commands to be empty, got %v", len(sync.Commands))
	}
}
