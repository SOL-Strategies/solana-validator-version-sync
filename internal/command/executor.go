package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
	"github.com/sol-strategies/solana-validator-version-sync/internal/rpc"
)

// Executor executes sync commands
type Executor struct {
	logger *log.Logger
}

// NewExecutor creates a new command executor
func NewExecutor() *Executor {
	return &Executor{
		logger: log.WithPrefix("command"),
	}
}

// CommandTemplateData represents the data available for command template interpolation
type CommandTemplateData struct {
	// Hostname of the validator
	Hostname string
	// Validator information
	Validator struct {
		RPCURL            string
		Role              string
		IdentityPublicKey string
	}
	// Sync information
	Sync struct {
		Client                  string
		FromVersion             string
		ToVersion               string
		Role                    string
		Cluster                 string
		IsSFDPComplianceEnabled bool
	}
}

// ExecuteCommands executes a list of commands in order
func (e *Executor) ExecuteCommands(commands []config.Command, data CommandTemplateData) error {
	for i, cmd := range commands {
		if cmd.Disabled {
			e.logger.Info("skipping disabled command", "index", i, "name", cmd.Name)
			continue
		}

		e.logger.Info("executing command", "index", i, "name", cmd.Name, "cmd", cmd.Cmd)

		if cmd.DryRun {
			e.logger.Info("dry run - would execute", "name", cmd.Name, "cmd", cmd.Cmd, "args", cmd.Args)
			continue
		}

		// Execute the command
		err := e.executeCommand(cmd, data)
		if err != nil {
			e.logger.Error("command failed", "name", cmd.Name, "error", err)

			if cmd.MustSucceed {
				return fmt.Errorf("command %s failed and must succeed: %w", cmd.Name, err)
			}

			if !cmd.AllowFailure {
				return fmt.Errorf("command %s failed: %w", cmd.Name, err)
			}

			e.logger.Warn("command failed but continuing", "name", cmd.Name, "error", err)
		} else {
			e.logger.Info("command completed successfully", "name", cmd.Name)
		}
	}

	return nil
}

// executeCommand executes a single command
func (e *Executor) executeCommand(cmd config.Command, data CommandTemplateData) error {
	// Render command arguments with template data
	renderedArgs, err := e.renderArgs(cmd.Args, data)
	if err != nil {
		return fmt.Errorf("failed to render command arguments: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute) // Commands can take a while
	defer cancel()

	// Create command
	execCmd := exec.CommandContext(ctx, cmd.Cmd, renderedArgs...)

	// Set up logging
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	e.logger.Info("running command", "cmd", cmd.Cmd, "args", renderedArgs)

	// Execute command
	return execCmd.Run()
}

// renderArgs renders command arguments using template interpolation
func (e *Executor) renderArgs(args []string, data CommandTemplateData) ([]string, error) {
	var renderedArgs []string

	for _, arg := range args {
		// Create template
		tmpl, err := template.New("arg").Parse(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}

		// Render template
		var rendered strings.Builder
		if err := tmpl.Execute(&rendered, data); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		renderedArgs = append(renderedArgs, rendered.String())
	}

	return renderedArgs, nil
}

// CreateCommandTemplateData creates template data from config and validator state
func CreateCommandTemplateData(cfg *config.Config, validatorState *rpc.ValidatorState, fromVersion, toVersion string) CommandTemplateData {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return CommandTemplateData{
		Hostname: hostname,
		Validator: struct {
			RPCURL            string
			Role              string
			IdentityPublicKey string
		}{
			RPCURL:            cfg.Validator.RPCURL,
			Role:              validatorState.Role,
			IdentityPublicKey: validatorState.IdentityPubkey,
		},
		Sync: struct {
			Client                  string
			FromVersion             string
			ToVersion               string
			Role                    string
			Cluster                 string
			IsSFDPComplianceEnabled bool
		}{
			Client:                  cfg.Validator.Client,
			FromVersion:             fromVersion,
			ToVersion:               toVersion,
			Role:                    validatorState.Role,
			Cluster:                 cfg.Cluster.Name,
			IsSFDPComplianceEnabled: cfg.Sync.EnableSFDPCompliance,
		},
	}
}
