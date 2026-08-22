package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/utils"
)

var AppVersion = "dev-build"

var (
	debugFlag bool
	forAIFlag bool
)

var rootCmd = &cobra.Command{
	Use:               "sharingan",
	Short:             "Manage long-lived EC2 workstations",
	Version:           AppVersion,
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupLogs() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime, NoColor: false}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debugFlag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		utils.GlobalDebugFlag = true
	}
	if forAIFlag {
		utils.GlobalForAIFlag = true
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&forAIFlag, "for-ai", false, "AI-friendly output (plain text, piped input)")
	rootCmd.MarkFlagsMutuallyExclusive("debug", "for-ai")

	cobra.OnInitialize(setupLogs)

	rootCmd.AddCommand(setConfigCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(teardownCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(modifyCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(costsCmd)
}
