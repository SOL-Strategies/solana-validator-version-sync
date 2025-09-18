package sync_commands

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var (
	stderrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("124"))
	stdoutStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("28"))
)

type ExecOptions struct {
	ExecLogger    *log.Logger
	CommandIndex  int
	CommandsCount int
	Disabled      bool
	AllowFailure  bool
	Cmd           string
	Args          []string
	Environment   map[string]string
	StreamOutput  bool
}

// Command is a command to run, contains valid templated strings
type Command struct {
	Name         string            `koanf:"name"`
	Disabled     bool              `koanf:"disabled"`
	AllowFailure bool              `koanf:"allow_failure"`
	Cmd          string            `koanf:"cmd"`
	Args         []string          `koanf:"args"`
	Environment  map[string]string `koanf:"environment"`
	StreamOutput bool              `koanf:"stream_output"`

	logger               *log.Logger
	cmdTemplate          *template.Template
	argsTemplates        []*template.Template
	environmentTemplates map[string]*template.Template
}

// CommandTemplateData represents the data available for command template interpolation
type CommandTemplateData struct {
	CommandIndex                int
	CommandsCount               int
	ValidatorClient             string
	ValidatorRPCURL             string
	ValidatorRole               string
	ValidatorRoleIsPassive      bool
	ValidatorRoleIsActive       bool
	ValidatorIdentityPublicKey  string
	ClusterName                 string
	VersionFrom                 string
	VersionTo                   string
	SyncIsSFDPComplianceEnabled bool
}

// NewCommand creates a new Command from a config
func (c *Command) Parse() (err error) {
	if c.Name == "" {
		return fmt.Errorf("command name is required")
	}

	// parse and store the command
	if c.Cmd == "" {
		return fmt.Errorf("command cmd is required")
	}
	c.cmdTemplate, err = template.New("cmd").Parse(c.Cmd)
	if err != nil {
		return fmt.Errorf("invalid golang template string: %w", err)
	}

	//  parse and store the arg templates
	c.argsTemplates = make([]*template.Template, len(c.Args))
	for j, arg := range c.Args {
		argTemplateName := fmt.Sprintf("arg[%d]", j)
		c.argsTemplates[j], err = template.New(argTemplateName).Parse(arg)
		if err != nil {
			return fmt.Errorf("invalid golang template string %s: %w", argTemplateName, err)
		}
	}

	// parse and store the environment templates
	c.environmentTemplates = make(map[string]*template.Template)
	for envName, envValue := range c.Environment {
		envTemplateName := fmt.Sprintf("env[%s]", envName)
		c.environmentTemplates[envName], err = template.New(envTemplateName).Parse(envValue)
		if err != nil {
			return fmt.Errorf("invalid golang template string %s: %w", envTemplateName, err)
		}
	}

	// create the logger
	c.logger = log.WithPrefix(fmt.Sprintf("command[%s]", c.Name)).
		With(
			"cmd", c.Cmd,
			"args", c.Args,
			"environment", c.Environment,
			"disabled", c.Disabled,
			"allow_failure", c.AllowFailure,
		)

	return nil
}

// ExecuteWithData executes the command with the provided template data
func (c *Command) ExecuteWithData(data CommandTemplateData) (err error) {
	var (
		compiledCmd         string
		compiledArgs        []string
		compiledEnvironment map[string]string
	)

	execLogger := log.WithPrefix(
		fmt.Sprintf("sync:commands[%d/%d %s]", data.CommandIndex+1, data.CommandsCount, c.Name),
	)

	execLogger.Debugf("executing command with data %+v", data)

	// compiled command
	cmdBuf := bytes.Buffer{}
	c.cmdTemplate.Execute(&cmdBuf, data)
	compiledCmd = cmdBuf.String()

	// compiled args
	compiledArgs = make([]string, len(c.argsTemplates))
	for _, argTemplate := range c.argsTemplates {
		argBuf := bytes.Buffer{}
		argTemplate.Execute(&argBuf, data)
		compiledArgs = append(compiledArgs, argBuf.String())
	}

	// compiled environment
	compiledEnvironment = make(map[string]string)
	for envName, envTemplate := range c.environmentTemplates {
		envBuf := bytes.Buffer{}
		envTemplate.Execute(&envBuf, data)
		compiledEnvironment[envName] = envBuf.String()
	}

	if c.Disabled {
		execLogger.Warn("command is disabled, skipping")
		return nil
	}

	return c.exec(ExecOptions{
		ExecLogger:    execLogger,
		CommandIndex:  data.CommandIndex,
		CommandsCount: data.CommandsCount,
		AllowFailure:  c.AllowFailure,
		Cmd:           compiledCmd,
		Args:          compiledArgs,
		Environment:   compiledEnvironment,
		StreamOutput:  c.StreamOutput,
	})
}

func (c *Command) exec(opts ExecOptions) (err error) {
	// doing something wrong here, but can't see it so make sure args exclude blank args
	sanitizedArgs := []string{}
	opts.ExecLogger.Debug("sanitizing args", "args", opts.Args)
	for _, arg := range opts.Args {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		sanitizedArgs = append(sanitizedArgs, arg)
	}
	sanitizedArgsJoined := strings.TrimSpace(strings.Join(sanitizedArgs, " "))
	opts.ExecLogger.Debug("sanitized args", "args", opts.Args, "sanitizedArgs", sanitizedArgs)

	opts.ExecLogger.With(
		"cmd", opts.Cmd,
		"args", sanitizedArgsJoined,
		"env", opts.Environment,
	).Info("running")

	// run it
	cmd := exec.Command(opts.Cmd, sanitizedArgs...)
	cmd.Env = opts.EnvironmentSlice()

	if opts.StreamOutput {
		// Capture stdout and stderr, then stream through logger
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe: %w", err)
		}

		// Start command
		err = cmd.Start()

		if err != nil && c.AllowFailure {
			opts.ExecLogger.Error("failed to start command with allow failure enabled - continuing", "error", err)
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to start command: %w", err)
		}

		// get the command pid (only after successful start)
		pid := cmd.Process.Pid
		opts.ExecLogger.Debug("command pid", "pid", pid)

		// Stream stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				opts.ExecLogger.Info(
					styledStreamOutputString("stdout", scanner.Text()),
				)
			}
		}()

		// Stream stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				opts.ExecLogger.Info(
					styledStreamOutputString("stderr", scanner.Text()),
				)
			}
		}()

		// Wait for command to complete
		err = cmd.Wait()
	} else {
		err = cmd.Run()
		// if failed and not allowed to fail, return error
		if err != nil && !opts.AllowFailure {
			opts.ExecLogger.Error("failed")
			return err
		}

		// if failed and allowed to fail say so and continue
		if err != nil && opts.AllowFailure {
			opts.ExecLogger.Warn(fmt.Sprintf("failed with sync.commands[%d].allow_failure=true - continuing", opts.CommandIndex), "error", err)
			return nil
		}
	}

	return nil
}

// EnvironmentSlice returns the environment variables as a slice of strings
func (o *ExecOptions) EnvironmentSlice() []string {
	env := make([]string, len(o.Environment))
	for k, v := range o.Environment {
		env = append(env, fmt.Sprintf("%s=%s", strings.TrimSpace(k), strings.TrimSpace(v)))
	}
	return env
}

func styledStreamOutputString(stream string, text string) string {
	// separater is faint gray, faint
	streamStyle := stdoutStyle
	if stream == "stderr" {
		streamStyle = stderrStyle
	}
	return fmt.Sprintf("%s %s", streamStyle.Render(">"), text)
}
