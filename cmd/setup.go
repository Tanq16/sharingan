package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/scaffold"
	u "github.com/tanq16/sharingan/utils"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create any missing scaffolding for the active account and region",
	Long:  "Create the VPC, subnet, internet gateway, route table, security group, and key pair that machines launch into. Safe to re-run.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := u.LoadConfig()
		if err != nil {
			u.PrintFatal("Failed to load the sharingan configuration", err)
		}
		clients, err := awsx.New(cmd.Context(), cfg.Profile, cfg.Region)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to reach AWS with profile %s in %s", cfg.Profile, cfg.Region), err)
		}

		u.PrintRunning(fmt.Sprintf("(Running) Scaffolding account %s in %s", clients.Account, clients.Region))
		var lines, created int
		scfg := scaffold.Config{Notify: func(e scaffold.Event) {
			u.PrintIndentedSuccess(fmt.Sprintf("%s: %s %s", e.Resource, e.Action, e.ID))
			lines++
			if e.Action == scaffold.Created {
				created++
			}
		}}

		err = scaffold.Setup(cmd.Context(), scfg, clients)
		u.ClearLines(lines + 1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Setup failed in %s", clients.Region), err)
		}
		u.PrintSuccess(fmt.Sprintf("Scaffolding ready in %s: %d created, %d already present", clients.Region, created, lines-created))
	},
}
