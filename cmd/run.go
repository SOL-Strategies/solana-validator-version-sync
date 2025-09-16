package cmd

import (
	"time"

	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/manager"
	"github.com/spf13/cobra"
)

var onIntervalDuration time.Duration

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "Start the Solana validator version sync manager",
	Long:          `Start the version sync manager to monitor the validator's version and sync it with the latest available versions.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		m, err := manager.NewFromConfig(loadedConfig)
		if err != nil {
			log.Fatal("failed to create sync manager", "error", err)
		}

		if onIntervalDuration != 0 {
			err = m.RunOnInterval(onIntervalDuration)
		} else {
			err = m.RunOnce()
		}

		if err != nil {
			log.Fatal("failed to run sync manager", "error", err)
		}
	},
}

func init() {
	runCmd.Flags().DurationVarP(&onIntervalDuration, "on-interval", "i", 0, "Run continuously at the specified interval (e.g., 1m, 30s, 1h). If not specified, runs once and exits.")
}
