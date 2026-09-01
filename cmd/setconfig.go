package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	u "github.com/tanq16/sharingan/utils"
)

var setConfigFlags struct {
	profile string
	region  string
}

var setConfigCmd = &cobra.Command{
	Use:   "set-config",
	Short: "Set the active AWS profile and region",
	Long:  "Set the active AWS profile and region. Anything not given as a flag is chosen interactively.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		profile := setConfigFlags.profile
		if profile == "" {
			profile = promptProfile()
		}
		region := setConfigFlags.region
		if region == "" {
			region = promptRegion()
		}

		cfg := &u.Config{Profile: profile, Region: region}
		if err := cfg.Save(); err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to write %s", u.ConfigPath()), err)
		}
		u.PrintSuccess(fmt.Sprintf("Active profile %s in region %s", profile, region))
	},
}

func promptProfile() string {
	profiles, err := u.AvailableProfiles()
	if err != nil {
		u.PrintFatal("Failed to read AWS profiles", err)
	}
	if len(profiles) == 0 {
		u.PrintFatal("No AWS profiles found in ~/.aws/config or ~/.aws/credentials", nil)
	}

	index, err := u.PromptSelect("Select an AWS profile:", profiles)
	if errors.Is(err, u.ErrNoTerminal) {
		u.PrintFatal("set-config needs --profile when there is no interactive terminal", nil)
	}
	if err != nil {
		u.PrintFatal("Failed to read the profile selection", err)
	}
	if index < 0 {
		u.PrintFatal("No profile selected", nil)
	}
	return profiles[index]
}

func promptRegion() string {
	region, err := u.PromptInput("AWS region:", "us-east-2")
	if errors.Is(err, u.ErrNoTerminal) {
		u.PrintFatal("set-config needs --region when there is no interactive terminal", nil)
	}
	if err != nil {
		u.PrintFatal("Failed to read the region", err)
	}
	if region == "" {
		u.PrintFatal("No region entered", nil)
	}
	return region
}

func init() {
	setConfigCmd.Flags().StringVarP(&setConfigFlags.profile, "profile", "p", "", "AWS profile name")
	setConfigCmd.Flags().StringVarP(&setConfigFlags.region, "region", "r", "", "AWS region")
}
