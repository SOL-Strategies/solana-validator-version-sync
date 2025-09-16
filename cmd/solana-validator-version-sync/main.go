package main

import (
	"os"

	"github.com/sol-strategies/solana-validator-version-sync/cmd"
)

func main() {
	// Set the version for Cobra (remove any trailing newlines)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
