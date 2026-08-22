package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/machine"
	u "github.com/tanq16/sharingan/utils"
)

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a stopped workstation",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := machineClients(ctx)

		u.PrintRunning(fmt.Sprintf("starting %s", name))
		started, err := machine.Start(ctx, clients, machine.StartConfig{Name: name})
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to start %s", name), err)
		}
		u.PrintSuccess(fmt.Sprintf("%s is running at %s", name, started.PublicIP))
	},
}
