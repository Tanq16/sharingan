package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/scaffold"
	u "github.com/tanq16/sharingan/utils"
)

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Delete the scaffolding for the active account and region",
	Long:  "Delete the VPC, subnet, internet gateway, route table, security group, and key pair. Refuses while managed instances exist.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := u.LoadConfig()
		if err != nil {
			u.PrintFatal("Failed to load the sharingan configuration", err)
		}
		clients, err := awsx.New(cmd.Context(), cfg.Profile, cfg.Region)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to reach AWS with profile %s in %s", cfg.Profile, cfg.Region), err)
		}

		u.PrintRunning(fmt.Sprintf("(Running) Tearing down account %s in %s", clients.Account, clients.Region))
		var lines int
		scfg := scaffold.Config{Notify: func(e scaffold.Event) {
			u.PrintIndentedSuccess(fmt.Sprintf("%s: %s %s", e.Resource, e.Action, e.ID))
			lines++
		}}

		err = scaffold.Teardown(cmd.Context(), scfg, clients)
		u.ClearLines(lines + 1)

		if blocked, ok := errors.AsType[*scaffold.InstancesExistError](err); ok {
			u.PrintError("Refusing to tear down while managed instances exist", err)
			for _, instance := range blocked.Instances {
				u.PrintIndentedError(instance, nil)
			}
			os.Exit(1)
		}
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Teardown failed in %s", clients.Region), err)
		}
		if lines == 0 {
			u.PrintInfo(fmt.Sprintf("No managed scaffolding found in %s", clients.Region))
			return
		}
		u.PrintSuccess(fmt.Sprintf("Teardown complete in %s: %d resources deleted", clients.Region, lines))
	},
}
