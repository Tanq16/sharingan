package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/machine"
	u "github.com/tanq16/sharingan/utils"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <name> [args...]",
	Short: "Open an SSH session on a workstation",
	Long:  "Open an SSH session on a workstation. Everything after the name is passed to the system ssh.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := activeClients(ctx)

		if err := machine.SSH(ctx, clients, machine.SSHConfig{Name: name, Args: args[1:]}); err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to connect to %s", name), err)
		}
	},
}

func init() {
	// Flag parsing stops at the machine name, so ssh's own flags reach ssh instead of cobra.
	sshCmd.Flags().SetInterspersed(false)
}
