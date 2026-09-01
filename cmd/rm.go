package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/machine"
	u "github.com/tanq16/sharingan/utils"
)

var rmFlags struct {
	yes bool
}

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Terminate a workstation and its root volume",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()

		if !rmFlags.yes {
			u.PrintWarn(fmt.Sprintf("%s and its root volume are deleted permanently", name), nil)
			typed, err := u.PromptInput("Retype the machine name to confirm:", name)
			if errors.Is(err, u.ErrNoTerminal) {
				u.PrintFatal("rm needs --yes when there is no interactive terminal", nil)
			}
			if err != nil {
				u.PrintFatal("Failed to read the confirmation", err)
			}
			if typed != name {
				u.PrintFatal(fmt.Sprintf("Confirmation %q does not match %q, nothing was deleted", typed, name), nil)
			}
		}

		clients := activeClients(ctx)
		u.PrintRunning(fmt.Sprintf("terminating %s", name))
		err := machine.Remove(ctx, clients, machine.RemoveConfig{Name: name})
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to terminate %s", name), err)
		}
		u.PrintSuccess(fmt.Sprintf("%s is terminated", name))
	},
}

func init() {
	rmCmd.Flags().BoolVar(&rmFlags.yes, "yes", false, "Skip the confirmation prompt")
}
