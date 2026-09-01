package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/inventory"
	u "github.com/tanq16/sharingan/utils"
)

var showFlags struct {
	origin bool
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "List the machines for the active account and region",
	Long:  "List the machines recorded in local state, or with --origin the machines AWS reports right now.",
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

		var machines []inventory.Machine
		if showFlags.origin {
			u.PrintRunning(fmt.Sprintf("Querying %s", clients.Region))
			machines, err = inventory.FromAPI(cmd.Context(), clients)
			u.ClearLines(1)
		} else {
			machines, err = inventory.FromState(clients.Account, clients.Region)
		}
		if err != nil {
			u.PrintFatal("Failed to read the machine inventory", err)
		}

		if len(machines) == 0 {
			u.PrintInfo(fmt.Sprintf("No machines in account %s region %s", clients.Account, clients.Region))
			return
		}
		u.PrintTable(inventory.Table(machines))
	},
}

func init() {
	showCmd.Flags().BoolVar(&showFlags.origin, "origin", false, "Read live from AWS instead of local state")
}
