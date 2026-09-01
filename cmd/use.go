package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	u "github.com/tanq16/sharingan/utils"
)

var useCmd = &cobra.Command{
	Use:   "use [profile]",
	Short: "Switch the active AWS profile",
	Long:  "Switch the active AWS profile. Without a name, the registered profiles are offered for selection.",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := u.LoadConfig()
		if err != nil {
			u.PrintFatal("Failed to read the sharingan configuration", err)
		}
		names := cfg.Names()
		if len(names) == 0 {
			u.PrintFatal("No AWS profile is registered, run `sharingan set-config` to add one", nil)
		}

		profile := ""
		if len(args) == 1 {
			profile = args[0]
		} else {
			labels := make([]string, len(names))
			for i, name := range names {
				labels[i] = fmt.Sprintf("%s (%s)", name, cfg.Region(name))
			}
			index, err := u.PromptSelect("Select a registered profile:", labels)
			if errors.Is(err, u.ErrNoTerminal) {
				u.PrintFatal("use needs a profile name when there is no interactive terminal", nil)
			}
			if err != nil {
				u.PrintFatal("Failed to read the profile selection", err)
			}
			if index < 0 {
				u.PrintFatal("No profile selected", nil)
			}
			profile = names[index]
		}

		if err := cfg.Use(profile); err != nil {
			u.PrintFatal("Failed to switch the active profile", err)
		}
		if err := cfg.Save(); err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to write %s", u.ConfigPath()), err)
		}
		u.PrintSuccess(fmt.Sprintf("Active profile %s in region %s", profile, cfg.Region(profile)))
	},
}
