package sync_commands

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/charmbracelet/log"
)

type ExecOptions struct {
	CommandIndex int
	Disabled     bool
	AllowFailure bool
	Cmd          string
	Args         []string
	Environment  map[string]string
	StreamOutput bool
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
	ValidatorClient             string
	ValidatorRPCURL             string
	ValidatorRole               string
	ValidatorRoleIsPassive      bool
	ValidatorRoleIsActive       bool
	ValidatorIdentityPublicKey  string
	ClusterName                 string
	Hostname                    string
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

	c.logger.Debugf("executing command with data %+v", data)

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
		c.logger.Info("command is disabled, skipping")
		return nil
	}

	return c.exec(ExecOptions{
		CommandIndex: data.CommandIndex,
		Disabled:     c.Disabled,
		AllowFailure: c.AllowFailure,
		Cmd:          compiledCmd,
		Args:         compiledArgs,
		Environment:  compiledEnvironment,
		StreamOutput: c.StreamOutput,
	})
}

func (c *Command) exec(opts ExecOptions) (err error) {
	execLogger := log.WithPrefix(fmt.Sprintf("sync.command[%d.%s]", opts.CommandIndex, c.Name))

	if opts.Disabled {
		execLogger.Info("command is disabled, skipping")
		return nil
	}

	// doing something wrong here, but can't see it so make sure args exclude blank args
	sanitizedArgs := []string{}
	execLogger.Debug("sanitizing args", "args", opts.Args)
	for _, arg := range opts.Args {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		sanitizedArgs = append(sanitizedArgs, arg)
	}
	sanitizedArgsJoined := strings.TrimSpace(strings.Join(sanitizedArgs, " "))
	execLogger.Debug("sanitized args", "args", opts.Args, "sanitizedArgs", sanitizedArgs)

	execLogger.With(
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
			execLogger.Error("failed to start command with allow failure enabled - continuing", "error", err)
			return nil
		}

		if err != nil && !opts.AllowFailure {
			execLogger.Error("failed to start command - not allowed to fail", "error", err)
			return err
		}

		// Stream stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				execLogger.Info(scanner.Text(), "stream", "stdout")
			}
		}()

		// Stream stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				execLogger.Warn(scanner.Text(), "stream", "stderr")
			}
		}()

		// Wait for command to complete
		err = cmd.Wait()
	} else {
		err = cmd.Run()
		// if failed and not allowed to fail, return error
		if err != nil && !opts.AllowFailure {
			execLogger.Error("command failed")
			return err
		}

		// if failed and allowed to fail say so and continue
		if err != nil && opts.AllowFailure {
			execLogger.Error("command failed with allow failure enabled - continuing", "error", err)
			return nil
		}
	}

	execLogger.Info("command completed successfully")

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
