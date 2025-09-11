package config

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

// Config represents the complete configuration
type Config struct {
	// Log configuration
	Log Log `koanf:"log"`
	// Validator is the local validator configuration
	Validator Validator `koanf:"validator"`
	// Cluster is the Solana cluster configuration
	Cluster Cluster `koanf:"cluster"`
	// Sync is the version sync configuration
	Sync Sync `koanf:"sync"`
	// File is the file that the config was loaded from
	File string `koanf:"-"`

	logger *log.Logger
}

// New creates a new Config
func New() (config *Config, err error) {
	config = &Config{
		logger: log.WithPrefix("config"),
	}
	return config, nil
}

// NewFromConfigFile creates a new Config from a config file path
func NewFromConfigFile(configFile string) (*Config, error) {
	// Create new config
	cfg, err := New()
	if err != nil {
		return nil, err
	}

	// Load from file
	if err := cfg.LoadFromFile(configFile); err != nil {
		return nil, err
	}

	// Initialize
	if err := cfg.Initialize(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromFile loads configuration from file into the struct
func (c *Config) LoadFromFile(filePath string) error {
	k := koanf.New(".")
	c.File = filePath

	// Load YAML config file
	if err := k.Load(file.Provider(c.File), yaml.Parser()); err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	// Unmarshal into this config struct
	if err := k.Unmarshal("", c); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	return nil
}

// Initialize processes and validates the loaded configuration
func (c *Config) Initialize() error {
	// Set defaults
	c.setDefaults()

	// load identity key pair files
	if err := c.Validator.Identities.Load(); err != nil {
		return err
	}

	// validate configuration (after identity files are loaded)
	if err := c.validate(); err != nil {
		return err
	}

	return nil
}

// validate validates the configuration
func (c *Config) validate() error {
	err := c.Log.Validate()
	if err != nil {
		return err
	}

	err = c.Validator.Validate()
	if err != nil {
		return err
	}

	err = c.Cluster.Validate()
	if err != nil {
		return err
	}

	err = c.Sync.Validate()
	if err != nil {
		return err
	}

	return nil
}

// setDefaults sets default values for configuration
func (c *Config) setDefaults() {
	c.Log.SetDefaults()
	c.Validator.SetDefaults()
	c.Cluster.SetDefaults()
	c.Sync.SetDefaults()
}
