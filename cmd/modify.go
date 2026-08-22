package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/machine"
	u "github.com/tanq16/sharingan/utils"
)

var modifyCmd = &cobra.Command{
	Use:   "modify <name>",
	Short: "Change the vCPU and memory of a workstation",
	Long:  "Change the vCPU and memory of a workstation. The architecture and the disk stay as they are.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := machineClients(ctx)

		current, err := clients.FindInstance(ctx, name)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to look up %s", name), err)
		}
		if current == nil {
			u.PrintFatal(fmt.Sprintf("No machine named %s in %s", name, clients.Region), nil)
		}

		resolver := machineResolver(ctx, clients)
		target := selectMachineShape(resolver, current.Arch)
		vcpu, memoryGB, arches, err := resolver.Details(ctx, target.InstanceType)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to read the details of %s", target.InstanceType), err)
		}

		u.PrintRunning(fmt.Sprintf("resizing %s from %s to %s", name, current.Type, target.InstanceType))
		modified, err := machine.Modify(ctx, clients, machine.ModifyConfig{
			Name:         name,
			InstanceType: target.InstanceType,
			VCPU:         vcpu,
			MemoryGB:     memoryGB,
			Arches:       arches,
		})
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to modify %s", name), err)
		}

		u.PrintSuccess(fmt.Sprintf("%s runs %s at %s", name, modified.InstanceType, modified.PublicIP))
		u.PrintInfo(fmt.Sprintf("%d vCPU, %d GB memory, %d GB disk", modified.VCPU, modified.MemoryGB, modified.DiskGB))
	},
}
