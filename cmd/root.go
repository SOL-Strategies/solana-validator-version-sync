package cmd

import (
	_ "embed"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
	"github.com/spf13/cobra"
)

//go:embed version.txt
var versionFile string

var version = strings.TrimSpace(strings.Split(versionFile, "\n")[0])

var (
	configFile   string
	logLevel     string
	loadedConfig *config.Config
)

var rootCmd = &cobra.Command{
	Use:     "solana-validator-version-sync",
	Short:   "Version sync manager for Solana validators",
	Version: version,
	Long: `Solana Validator Version Sync is a version synchronization manager for Solana validators.
It monitors the validator's current version and syncs it with the latest available versions.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration
		var err error
		loadedConfig, err = config.NewFromConfigFile(configFile)
		if err != nil {
			log.Fatal("failed to load configuration", "error", err)
		}

		loadedConfig.Log.ConfigureWithLevelString(logLevel)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add global flags here
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "~/solana-validator-version-sync/config.yaml", "Path to configuration file (default: ~/solana-validator-version-sync/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "Log level (debug, info, warn, error, fatal) - overrides config.yaml log.level if specified")

	// Add subcommands here
	rootCmd.AddCommand(runCmd)
}
