package main

import (
	"fmt"
	"os"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"

	cfgFile string
	verbose bool
	logger  *logrus.Logger
	cfg     *config.Config
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crisk",
	Short: "CodeRisk - Lightning-fast risk assessment for code changes",
	Long: `CodeRisk performs sub-5-second risk analysis on your code changes,
helping you catch potential issues before they reach production.`,
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger
		logger = logrus.New()
		if verbose {
			logger.SetLevel(logrus.DebugLevel)
		} else {
			logger.SetLevel(logrus.InfoLevel)
		}

		// Load configuration
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			logger.WithError(err).Warn("Failed to load config, using defaults")
			cfg = config.Default()
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .coderisk/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Set custom version template
	rootCmd.SetVersionTemplate(`CodeRisk {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(initLocalCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(incidentCmd)
}
