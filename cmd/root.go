// Package cmd contains command execution logic.
package cmd

import (
	"context"
	"os"
	"strings"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/kalverra/agents/cmd/skills"
	"github.com/kalverra/agents/internal/config"
)

var cfg = &config.Config{}

var rootCmd = &cobra.Command{
	Use:   "agents",
	Short: "Helper CLI for AI agent workflows",
	Long:  `Helper CLI for AI agent workflows`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		var err error
		cfg, err = config.Load(config.WithFlags(cmd.Flags()))
		return err
	},
}

func init() {
	// Handle kebab-case flags as snake_case env vars for clean config
	rootCmd.SetGlobalNormalizationFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		return pflag.NormalizedName(strings.ReplaceAll(name, "-", "_"))
	})
	rootCmd.PersistentFlags().
		StringVarP(&cfg.LogLevel, "log-level", "l", config.DefaultLogLevel, "Log level (env: LOG_LEVEL)")

	rootCmd.AddCommand(skills.Cmd)
}

// Execute runs the root command.
func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
