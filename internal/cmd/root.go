package cmd

import (
	"os"

	"github.com/nipunap/kim/internal/config"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/internal/ui"

	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	debug       bool
	interactive bool
)

// Execute executes the root command
func Execute(cfg *config.Config, log *logger.Logger) error {
	rootCmd := NewRootCmd(cfg, log)
	return rootCmd.Execute()
}

// NewRootCmd creates the root command
func NewRootCmd(cfg *config.Config, log *logger.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kim",
		Short: "Kim - Kafka Management Tool",
		Long: `Kim is a powerful command-line interface for managing Kafka and MSK clusters.
It provides an intuitive way to interact with Kafka topics, consumer groups, and messages
with support for both regular Kafka and AWS MSK clusters.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel("debug")
				log.Debug("Debug logging enabled")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if interactive {
				// Start interactive mode
				if err := runInteractiveMode(cfg, log); err != nil {
					log.Error("Interactive mode failed", "error", err)
					os.Exit(1)
				}
				return
			}

			// Show help if no command provided
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.github.com/nipunap/kim/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "run in interactive mode")

	// Add subcommands
	rootCmd.AddCommand(NewTopicCmd(cfg, log))
	rootCmd.AddCommand(NewGroupCmd(cfg, log))
	rootCmd.AddCommand(NewMessageCmd(cfg, log))
	rootCmd.AddCommand(NewProfileCmd(cfg, log))

	return rootCmd
}

// runInteractiveMode starts the interactive mode
func runInteractiveMode(cfg *config.Config, log *logger.Logger) error {
	ui := ui.NewInteractiveMode(cfg, log)
	return ui.Run()
}
