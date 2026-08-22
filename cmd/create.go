package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/machine"
	"github.com/tanq16/sharingan/internal/pricing"
	"github.com/tanq16/sharingan/internal/sizing"
	u "github.com/tanq16/sharingan/utils"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Launch a workstation",
	Long:  "Launch a workstation. Architecture, size, class, and disk are chosen interactively.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := machineClients(ctx)
		resolver := machineResolver(ctx, clients)

		target := selectMachineShape(resolver, "")
		disk := selectMachineDisk()

		u.PrintRunning(fmt.Sprintf("launching %s as %s", name, target.InstanceType))
		created, err := machine.Create(ctx, clients, machine.CreateConfig{
			Name:         name,
			InstanceType: target.InstanceType,
			Arch:         target.Arch,
			VCPU:         target.VCPU,
			MemoryGB:     target.MemoryGB,
			DiskGB:       disk,
		})
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to create %s", name), err)
		}

		u.PrintSuccess(fmt.Sprintf("%s is running at %s", name, created.PublicIP))
		u.PrintInfo(fmt.Sprintf("%s on %s, %d vCPU, %d GB memory, %d GB disk",
			created.InstanceType, created.Arch, created.VCPU, created.MemoryGB, created.DiskGB))
		u.PrintInfo(fmt.Sprintf("bootstrap is still running, follow it with `sharingan status %s`", name))
	},
}

func machineClients(ctx context.Context) *awsx.Clients {
	cfg, err := u.LoadConfig()
	if err != nil {
		u.PrintFatal("Failed to read the sharingan configuration", err)
	}
	clients, err := awsx.New(ctx, cfg.Profile, cfg.Region)
	if err != nil {
		u.PrintFatal(fmt.Sprintf("Failed to reach AWS as %s in %s", cfg.Profile, cfg.Region), err)
	}
	return clients
}

func machineResolver(ctx context.Context, clients *awsx.Clients) *sizing.Resolver {
	u.PrintRunning("reading prices and instance offerings")
	prices, err := pricing.Fetch(ctx, clients)
	if err != nil {
		u.ClearLines(1)
		u.PrintFatal("Failed to read on-demand prices", err)
	}
	resolver, err := sizing.NewResolver(ctx, clients, prices)
	u.ClearLines(1)
	if err != nil {
		u.PrintFatal("Failed to read instance offerings", err)
	}
	return resolver
}

// A non-empty arch fixes the architecture, which is how modify keeps the one its root volume holds.
func selectMachineShape(resolver *sizing.Resolver, arch string) *sizing.Resolved {
	if arch == "" {
		arches := []string{sizing.ArchARM, sizing.ArchX86}
		labels := []string{sizing.ArchARM + " (cheaper at every shape)", sizing.ArchX86}
		arch = arches[machineChoice("Architecture:", labels)]
	}

	shapes := sizing.Shapes()
	shapeLabels := make([]string, len(shapes))
	for i, shape := range shapes {
		shapeLabels[i] = fmt.Sprintf("%d vCPU / %d GB", shape.VCPU, shape.MemoryGB)
	}
	shape := shapes[machineChoice("Size:", shapeLabels)]

	var offered []*sizing.Resolved
	var classLabels []string
	for _, class := range []sizing.Class{sizing.Burstable, sizing.Dedicated} {
		// A shape with no type in that class resolves to nothing, and must not be offered.
		resolved, err := resolver.Resolve(shape, arch, class)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to resolve a %s type for %d vCPU / %d GB on %s", class, shape.VCPU, shape.MemoryGB, arch), err)
		}
		if resolved == nil {
			continue
		}
		offered = append(offered, resolved)
		classLabels = append(classLabels, fmt.Sprintf("%s: %s at $%.4f per hour", class, resolved.InstanceType, resolved.HourlyUSD))
	}
	if len(offered) == 0 {
		u.PrintFatal(fmt.Sprintf("No instance type offers %d vCPU / %d GB on %s", shape.VCPU, shape.MemoryGB, arch), nil)
	}
	return offered[machineChoice("Class:", classLabels)]
}

func selectMachineDisk() int {
	sizes := sizing.DiskSizes()
	labels := make([]string, len(sizes))
	for i, size := range sizes {
		labels[i] = fmt.Sprintf("%d GB", size)
	}
	return sizes[machineChoice("Disk:", labels)]
}

func machineChoice(label string, options []string) int {
	index, err := u.PromptSelect(label, options)
	if err != nil {
		u.PrintFatal("Failed to read the selection", err)
	}
	if index < 0 {
		u.PrintFatal("Nothing selected", nil)
	}
	return index
}
