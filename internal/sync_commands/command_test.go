package sync_commands

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecOptions_StructFields(t *testing.T) {
	opts := ExecOptions{
		CommandIndex:       1,
		Disabled:           false,
		AllowFailure:       true,
		Cmd:                "echo",
		Args:               []string{"hello", "world"},
		Environment:        map[string]string{"TEST": "value"},
		InheritEnvironment: true,
		StreamOutput:       true,
	}

	if opts.CommandIndex != 1 {
		t.Errorf("Expected CommandIndex to be 1, got %d", opts.CommandIndex)
	}
	if opts.Disabled != false {
		t.Errorf("Expected Disabled to be false, got %v", opts.Disabled)
	}
	if opts.AllowFailure != true {
		t.Errorf("Expected AllowFailure to be true, got %v", opts.AllowFailure)
	}
	if opts.Cmd != "echo" {
		t.Errorf("Expected Cmd to be echo, got %s", opts.Cmd)
	}
	if len(opts.Args) != 2 {
		t.Errorf("Expected Args length to be 2, got %d", len(opts.Args))
	}
	if opts.Args[0] != "hello" {
		t.Errorf("Expected first arg to be hello, got %s", opts.Args[0])
	}
	if opts.Args[1] != "world" {
		t.Errorf("Expected second arg to be world, got %s", opts.Args[1])
	}
	if opts.Environment["TEST"] != "value" {
		t.Errorf("Expected Environment TEST to be value, got %s", opts.Environment["TEST"])
	}
	if opts.InheritEnvironment != true {
		t.Errorf("Expected InheritEnvironment to be true, got %v", opts.InheritEnvironment)
	}
	if opts.StreamOutput != true {
		t.Errorf("Expected StreamOutput to be true, got %v", opts.StreamOutput)
	}
}

func TestCommand_StructFields(t *testing.T) {
	cmd := Command{
		Name:               "test-command",
		Disabled:           false,
		AllowFailure:       true,
		Cmd:                "echo",
		Args:               []string{"{{.VersionTo}}"},
		Environment:        map[string]string{"CLUSTER": "{{.ClusterName}}"},
		InheritEnvironment: true,
		StreamOutput:       true,
	}

	if cmd.Name != "test-command" {
		t.Errorf("Expected Name to be test-command, got %s", cmd.Name)
	}
	if cmd.Disabled != false {
		t.Errorf("Expected Disabled to be false, got %v", cmd.Disabled)
	}
	if cmd.AllowFailure != true {
		t.Errorf("Expected AllowFailure to be true, got %v", cmd.AllowFailure)
	}
	if cmd.Cmd != "echo" {
		t.Errorf("Expected Cmd to be echo, got %s", cmd.Cmd)
	}
	if len(cmd.Args) != 1 {
		t.Errorf("Expected Args length to be 1, got %d", len(cmd.Args))
	}
	if cmd.Args[0] != "{{.VersionTo}}" {
		t.Errorf("Expected first arg to be {{.VersionTo}}, got %s", cmd.Args[0])
	}
	if cmd.Environment["CLUSTER"] != "{{.ClusterName}}" {
		t.Errorf("Expected Environment CLUSTER to be {{.ClusterName}}, got %s", cmd.Environment["CLUSTER"])
	}
	if cmd.InheritEnvironment != true {
		t.Errorf("Expected InheritEnvironment to be true, got %v", cmd.InheritEnvironment)
	}
	if cmd.StreamOutput != true {
		t.Errorf("Expected StreamOutput to be true, got %v", cmd.StreamOutput)
	}
}

func TestCommandTemplateData_StructFields(t *testing.T) {
	data := CommandTemplateData{
		CommandIndex:                1,
		ValidatorClient:             "agave",
		ValidatorRPCURL:             "http://localhost:8899",
		ValidatorRole:               "active",
		ValidatorRoleIsPassive:      false,
		ValidatorRoleIsActive:       true,
		ValidatorIdentityPublicKey:  "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM",
		ClusterName:                 "mainnet-beta",
		VersionFrom:                 "1.17.0",
		VersionTo:                   "1.18.0",
		SyncIsSFDPComplianceEnabled: true,
	}

	if data.CommandIndex != 1 {
		t.Errorf("Expected CommandIndex to be 1, got %d", data.CommandIndex)
	}
	if data.ValidatorClient != "agave" {
		t.Errorf("Expected ValidatorClient to be agave, got %s", data.ValidatorClient)
	}
	if data.ValidatorRPCURL != "http://localhost:8899" {
		t.Errorf("Expected ValidatorRPCURL to be http://localhost:8899, got %s", data.ValidatorRPCURL)
	}
	if data.ValidatorRole != "active" {
		t.Errorf("Expected ValidatorRole to be active, got %s", data.ValidatorRole)
	}
	if data.ValidatorRoleIsPassive != false {
		t.Errorf("Expected ValidatorRoleIsPassive to be false, got %v", data.ValidatorRoleIsPassive)
	}
	if data.ValidatorRoleIsActive != true {
		t.Errorf("Expected ValidatorRoleIsActive to be true, got %v", data.ValidatorRoleIsActive)
	}
	if data.ValidatorIdentityPublicKey != "9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM" {
		t.Errorf("Expected ValidatorIdentityPublicKey to be 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM, got %s", data.ValidatorIdentityPublicKey)
	}
	if data.ClusterName != "mainnet-beta" {
		t.Errorf("Expected ClusterName to be mainnet-beta, got %s", data.ClusterName)
	}
	if data.VersionFrom != "1.17.0" {
		t.Errorf("Expected VersionFrom to be 1.17.0, got %s", data.VersionFrom)
	}
	if data.VersionTo != "1.18.0" {
		t.Errorf("Expected VersionTo to be 1.18.0, got %s", data.VersionTo)
	}
	if data.SyncIsSFDPComplianceEnabled != true {
		t.Errorf("Expected SyncIsSFDPComplianceEnabled to be true, got %v", data.SyncIsSFDPComplianceEnabled)
	}
}

func TestCommand_Parse(t *testing.T) {
	tests := []struct {
		name    string
		command Command
		wantErr bool
	}{
		{
			name: "valid command with templates",
			command: Command{
				Name: "test-command",
				Cmd:  "echo",
				Args: []string{"{{.VersionTo}}", "{{.ClusterName}}"},
				Environment: map[string]string{
					"CLUSTER": "{{.ClusterName}}",
					"VERSION": "{{.VersionTo}}",
				},
			},
			wantErr: false,
		},
		{
			name: "valid command without templates",
			command: Command{
				Name: "simple-command",
				Cmd:  "echo",
				Args: []string{"hello", "world"},
				Environment: map[string]string{
					"TEST": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "missing command name",
			command: Command{
				Cmd: "echo",
			},
			wantErr: true,
		},
		{
			name: "missing command cmd",
			command: Command{
				Name: "test-command",
			},
			wantErr: true,
		},
		{
			name: "invalid template in cmd",
			command: Command{
				Name: "test-command",
				Cmd:  "echo {{.InvalidTemplate",
			},
			wantErr: true,
		},
		{
			name: "invalid template in args",
			command: Command{
				Name: "test-command",
				Cmd:  "echo",
				Args: []string{"{{.InvalidTemplate"},
			},
			wantErr: true,
		},
		{
			name: "invalid template in environment",
			command: Command{
				Name: "test-command",
				Cmd:  "echo",
				Environment: map[string]string{
					"TEST": "{{.InvalidTemplate",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.command.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Check that templates were parsed correctly
				if tt.command.cmdTemplate == nil {
					t.Error("Parse() should set cmdTemplate")
				}
				if tt.command.argsTemplates == nil {
					t.Error("Parse() should set argsTemplates")
				}
				if tt.command.environmentTemplates == nil {
					t.Error("Parse() should set environmentTemplates")
				}
				if tt.command.logger == nil {
					t.Error("Parse() should set logger")
				}
			}
		})
	}
}

func TestCommand_ExecuteWithData(t *testing.T) {
	// Skip if not on Unix-like system (for echo command)
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tests := []struct {
		name       string
		command    Command
		data       CommandTemplateData
		wantErr    bool
		expectSkip bool
	}{
		{
			name: "successful command execution",
			command: Command{
				Name: "test-echo",
				Cmd:  "echo",
				Args: []string{"{{.VersionTo}}"},
			},
			data: CommandTemplateData{
				VersionTo: "1.18.0",
			},
			wantErr:    false,
			expectSkip: false,
		},
		{
			name: "disabled command",
			command: Command{
				Name:     "disabled-command",
				Cmd:      "echo",
				Args:     []string{"should not run"},
				Disabled: true,
			},
			data:       CommandTemplateData{},
			wantErr:    false,
			expectSkip: true,
		},
		{
			name: "command with environment variables",
			command: Command{
				Name: "env-command",
				Cmd:  "echo",
				Args: []string{"$CLUSTER"},
				Environment: map[string]string{
					"CLUSTER": "{{.ClusterName}}",
				},
			},
			data: CommandTemplateData{
				ClusterName: "testnet",
			},
			wantErr:    false,
			expectSkip: false,
		},
		{
			name: "command with complex templates",
			command: Command{
				Name: "complex-command",
				Cmd:  "echo",
				Args: []string{"{{.ValidatorClient}}", "{{.VersionFrom}}", "to", "{{.VersionTo}}"},
				Environment: map[string]string{
					"CLUSTER": "{{.ClusterName}}",
					"ROLE":    "{{.ValidatorRole}}",
				},
			},
			data: CommandTemplateData{
				ValidatorClient: "agave",
				VersionFrom:     "1.17.0",
				VersionTo:       "1.18.0",
				ClusterName:     "mainnet-beta",
				ValidatorRole:   "active",
			},
			wantErr:    false,
			expectSkip: false,
		},
		{
			name: "command that fails but allows failure",
			command: Command{
				Name:         "failing-command",
				Cmd:          "nonexistent-command-that-should-fail",
				Args:         []string{},
				AllowFailure: true,
			},
			data:       CommandTemplateData{},
			wantErr:    false, // Should not return error due to AllowFailure
			expectSkip: false,
		},
		{
			name: "command that fails and does not allow failure",
			command: Command{
				Name:         "failing-command",
				Cmd:          "nonexistent-command-that-should-fail",
				Args:         []string{},
				AllowFailure: false,
			},
			data:       CommandTemplateData{},
			wantErr:    true, // Should return error
			expectSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the command first
			err := tt.command.Parse()
			if err != nil {
				t.Fatalf("Parse() failed: %v", err)
			}

			// Execute the command
			err = tt.command.ExecuteWithData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteWithData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommand_ExecuteWithData_StreamOutput(t *testing.T) {
	// Skip if not on Unix-like system
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	command := Command{
		Name:         "streaming-command",
		Cmd:          "echo",
		Args:         []string{"{{.VersionTo}}"},
		StreamOutput: true,
	}

	data := CommandTemplateData{
		VersionTo: "1.18.0",
	}

	// Parse the command first
	err := command.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Execute the command
	err = command.ExecuteWithData(data)
	if err != nil {
		t.Errorf("ExecuteWithData() error = %v", err)
	}
}

func TestExecOptions_EnvironmentSlice(t *testing.T) {
	testsEnvMap := func(t *testing.T, env []string) map[string]string {
		t.Helper()

		result := make(map[string]string, len(env))
		for _, envVar := range env {
			k, v, ok := strings.Cut(envVar, "=")
			if !ok {
				t.Fatalf("EnvironmentSlice() returned invalid env var: %q", envVar)
			}
			if _, exists := result[k]; exists {
				t.Fatalf("EnvironmentSlice() returned duplicate key: %q", k)
			}
			result[k] = v
		}

		return result
	}

	tests := []struct {
		name     string
		opts     ExecOptions
		setup    func(t *testing.T)
		expected map[string]string
		exact    bool
	}{
		{
			name: "empty environment without inheritance",
			opts: ExecOptions{
				Environment: map[string]string{},
			},
			expected: map[string]string{},
			exact:    true,
		},
		{
			name: "single environment variable without inheritance",
			opts: ExecOptions{
				Environment: map[string]string{
					"TEST": "value",
				},
			},
			expected: map[string]string{"TEST": "value"},
			exact:    true,
		},
		{
			name: "multiple environment variables without inheritance",
			opts: ExecOptions{
				Environment: map[string]string{
					"CLUSTER": "mainnet-beta",
					"VERSION": "1.18.0",
					"ROLE":    "active",
				},
			},
			expected: map[string]string{
				"CLUSTER": "mainnet-beta",
				"VERSION": "1.18.0",
				"ROLE":    "active",
			},
			exact: true,
		},
		{
			name: "environment variables with spaces without inheritance",
			opts: ExecOptions{
				Environment: map[string]string{
					"KEY1": " value1 ",
					"KEY2": "value2",
				},
			},
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			exact: true,
		},
		{
			name: "inherits parent environment when enabled",
			opts: ExecOptions{
				Environment:        map[string]string{},
				InheritEnvironment: true,
			},
			setup: func(t *testing.T) {
				t.Setenv("SVVS_TEST_INHERITED", "parent")
			},
			expected: map[string]string{
				"SVVS_TEST_INHERITED": "parent",
			},
		},
		{
			name: "adds explicit environment on top of inherited values",
			opts: ExecOptions{
				Environment: map[string]string{
					"SVVS_TEST_ADDED": "child",
				},
				InheritEnvironment: true,
			},
			setup: func(t *testing.T) {
				t.Setenv("SVVS_TEST_BASE", "parent")
			},
			expected: map[string]string{
				"SVVS_TEST_BASE":  "parent",
				"SVVS_TEST_ADDED": "child",
			},
		},
		{
			name: "explicit environment overrides inherited values",
			opts: ExecOptions{
				Environment: map[string]string{
					"SVVS_TEST_OVERRIDE": " child ",
				},
				InheritEnvironment: true,
			},
			setup: func(t *testing.T) {
				t.Setenv("SVVS_TEST_OVERRIDE", "parent")
			},
			expected: map[string]string{
				"SVVS_TEST_OVERRIDE": "child",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			result := testsEnvMap(t, tt.opts.EnvironmentSlice())

			if tt.exact && len(result) != len(tt.expected) {
				t.Errorf("EnvironmentSlice() length = %d, want %d", len(result), len(tt.expected))
			}

			for expectedKey, expectedValue := range tt.expected {
				actualValue, found := result[expectedKey]
				if !found {
					t.Errorf("EnvironmentSlice() missing expected key: %s", expectedKey)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("EnvironmentSlice() value for %s = %q, want %q", expectedKey, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestCommand_ExecuteWithData_RealCommand(t *testing.T) {
	// Skip if not on Unix-like system
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	// Test with a real command that we know exists
	command := Command{
		Name: "real-command-test",
		Cmd:  "echo",
		Args: []string{"Hello", "{{.VersionTo}}"},
	}

	data := CommandTemplateData{
		VersionTo: "1.18.0",
	}

	// Parse the command first
	err := command.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Execute the command
	err = command.ExecuteWithData(data)
	if err != nil {
		t.Errorf("ExecuteWithData() error = %v", err)
	}
}

func TestCommand_ExecuteWithData_Timeout(t *testing.T) {
	// Skip if not on Unix-like system
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	// Test with a command that takes some time
	command := Command{
		Name: "sleep-command",
		Cmd:  "sleep",
		Args: []string{"1"}, // Sleep for 1 second
	}

	data := CommandTemplateData{}

	// Parse the command first
	err := command.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Execute the command and measure time
	start := time.Now()
	err = command.ExecuteWithData(data)
	duration := time.Since(start)

	// If sleep command is not available, skip the test
	if err != nil {
		t.Skipf("Sleep command not available, skipping timeout test: %v", err)
	}

	// Should take at least 1 second
	if duration < time.Second {
		t.Errorf("Command should have taken at least 1 second, took %v", duration)
	}
}

func TestCommand_ExecuteWithData_InvalidCommand(t *testing.T) {
	command := Command{
		Name: "invalid-command",
		Cmd:  "this-command-does-not-exist-12345",
		Args: []string{},
	}

	data := CommandTemplateData{}

	// Parse the command first
	err := command.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Execute the command - should fail
	err = command.ExecuteWithData(data)
	if err == nil {
		t.Error("ExecuteWithData() should have failed for invalid command")
	}
}

func TestCommand_ExecuteWithData_InvalidCommandWithAllowFailure(t *testing.T) {
	command := Command{
		Name:         "invalid-command-with-allow-failure",
		Cmd:          "this-command-does-not-exist-12345",
		Args:         []string{},
		AllowFailure: true,
	}

	data := CommandTemplateData{}

	// Parse the command first
	err := command.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Execute the command - should not fail due to AllowFailure
	err = command.ExecuteWithData(data)
	if err != nil {
		t.Errorf("ExecuteWithData() should not have failed with AllowFailure=true, got error: %v", err)
	}
}
