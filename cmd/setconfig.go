package cmd

import (
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
	Long:  "Set the active AWS profile and region. Run without flags to pick them interactively.",
	Run: func(cmd *cobra.Command, args []string) {
		profile, region := setConfigFlags.profile, setConfigFlags.region
		if profile == "" && region == "" {
			profile, region = promptProfileAndRegion()
		}
		if profile == "" {
			u.PrintFatal("--profile is required", nil)
		}
		if region == "" {
			u.PrintFatal("--region is required", nil)
		}

		cfg := &u.Config{Profile: profile, Region: region}
		if err := cfg.Save(); err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to write %s", u.ConfigPath()), err)
		}
		u.PrintSuccess(fmt.Sprintf("Active profile %s in region %s", profile, region))
	},
}

func promptProfileAndRegion() (string, string) {
	profiles, err := u.AvailableProfiles()
	if err != nil {
		u.PrintFatal("Failed to read AWS profiles", err)
	}
	if len(profiles) == 0 {
		u.PrintFatal("No AWS profiles found in ~/.aws/config or ~/.aws/credentials", nil)
	}

	index, err := u.PromptSelect("Select an AWS profile:", profiles)
	if err != nil {
		u.PrintFatal("Failed to read the profile selection", err)
	}
	if index < 0 {
		u.PrintFatal("No profile selected", nil)
	}

	region, err := u.PromptInput("AWS region:", "us-east-2")
	if err != nil {
		u.PrintFatal("Failed to read the region", err)
	}
	return profiles[index], region
}

func init() {
	setConfigCmd.Flags().StringVarP(&setConfigFlags.profile, "profile", "p", "", "AWS profile name")
	setConfigCmd.Flags().StringVarP(&setConfigFlags.region, "region", "r", "", "AWS region")
}
