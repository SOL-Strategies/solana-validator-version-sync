package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/sol-strategies/solana-validator-version-sync/internal/sync"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "Start the Solana validator version sync manager",
	Long:          `Start the version sync manager to monitor the validator's version and sync it with the latest available versions.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Start the sync manager with the loaded config
		manager := sync.NewManager(sync.NewManagerOptions{
			Cfg: loadedConfig,
		})
		err := manager.Run()
		if err != nil {
			log.Fatal("failed to run sync manager", "error", err)
		}
	},
}
