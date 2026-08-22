package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/machine"
	u "github.com/tanq16/sharingan/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Report bootstrap progress over SSH",
	Long:  "Report bootstrap progress over SSH. It answers only once port 22 is open.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := machineClients(ctx)

		u.PrintRunning(fmt.Sprintf("probing %s over ssh", name))
		report, err := machine.Status(ctx, clients, machine.StatusConfig{Name: name})
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to probe %s", name), err)
		}
		u.PrintGeneric(report)
	},
}
