package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/machine"
	u "github.com/tanq16/sharingan/utils"
)

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running workstation",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := activeClients(ctx)

		u.PrintRunning(fmt.Sprintf("stopping %s", name))
		err := machine.Stop(ctx, clients, machine.StopConfig{Name: name})
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to stop %s", name), err)
		}
		u.PrintSuccess(fmt.Sprintf("%s is stopped, and its public address is released", name))
	},
}
