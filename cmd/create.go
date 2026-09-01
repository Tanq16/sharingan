package cmd

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/machine"
	"github.com/tanq16/sharingan/internal/pricing"
	"github.com/tanq16/sharingan/internal/sizing"
	u "github.com/tanq16/sharingan/utils"
)

var createFlags struct {
	arch  archFlag
	shape shapeFlag
	class classFlag
	disk  diskFlag
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Launch a workstation",
	Long:  "Launch a workstation. Anything not given as a flag is chosen interactively.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		ctx := cmd.Context()
		clients := activeClients(ctx)
		resolver := machineResolver(ctx, clients)

		target := resolveShape(resolver, "create", string(createFlags.arch), string(createFlags.shape), string(createFlags.class))
		disk := resolveDisk("create", int(createFlags.disk))

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

func resolveShape(resolver *sizing.Resolver, cmdName, arch, shapeName, class string) *sizing.Resolved {
	if arch == "" {
		arches := []string{sizing.ArchARM, sizing.ArchX86}
		labels := []string{sizing.ArchARM + " (cheaper at every shape)", sizing.ArchX86}
		arch = arches[machineChoice("Architecture:", labels, cmdName+" needs --arch when there is no interactive terminal")]
	}

	shapes := sizing.Shapes()
	var shape sizing.Shape
	if shapeName != "" {
		shape = shapes[slices.Index(shapeValues(), shapeName)]
	} else {
		labels := make([]string, len(shapes))
		for i, s := range shapes {
			labels[i] = fmt.Sprintf("%d vCPU / %d GB", s.VCPU, s.MemoryGB)
		}
		shape = shapes[machineChoice("Size:", labels, cmdName+" needs --shape when there is no interactive terminal")]
	}

	var offered []*sizing.Resolved
	var classLabels []string
	for _, c := range []sizing.Class{sizing.Burstable, sizing.Dedicated} {
		// A shape with no type in that class resolves to nothing, and must not be offered.
		resolved, err := resolver.Resolve(shape, arch, c)
		if err != nil {
			u.PrintFatal(fmt.Sprintf("Failed to resolve a %s type for %d vCPU / %d GB on %s", c, shape.VCPU, shape.MemoryGB, arch), err)
		}
		if resolved == nil {
			continue
		}
		offered = append(offered, resolved)
		classLabels = append(classLabels, fmt.Sprintf("%s: %s at $%.4f per hour", c, resolved.InstanceType, resolved.HourlyUSD))
	}
	if len(offered) == 0 {
		u.PrintFatal(fmt.Sprintf("No instance type offers %d vCPU / %d GB on %s", shape.VCPU, shape.MemoryGB, arch), nil)
	}

	if class != "" {
		i := slices.IndexFunc(offered, func(r *sizing.Resolved) bool { return string(r.Class) == class })
		if i < 0 {
			u.PrintFatal(fmt.Sprintf("No %s instance type offers %d vCPU / %d GB on %s", class, shape.VCPU, shape.MemoryGB, arch), nil)
		}
		return offered[i]
	}
	return offered[machineChoice("Class:", classLabels, cmdName+" needs --class when there is no interactive terminal")]
}

func resolveDisk(cmdName string, gb int) int {
	if gb > 0 {
		return gb
	}
	sizes := sizing.DiskSizes()
	labels := make([]string, len(sizes))
	for i, size := range sizes {
		labels[i] = fmt.Sprintf("%d GB", size)
	}
	return sizes[machineChoice("Disk:", labels, cmdName+" needs --disk when there is no interactive terminal")]
}

func machineChoice(label string, options []string, noTerminalHint string) int {
	index, err := u.PromptSelect(label, options)
	if errors.Is(err, u.ErrNoTerminal) {
		u.PrintFatal(noTerminalHint, nil)
	}
	if err != nil {
		u.PrintFatal("Failed to read the selection", err)
	}
	if index < 0 {
		u.PrintFatal("Nothing selected", nil)
	}
	return index
}

type archFlag string

func (a *archFlag) String() string { return string(*a) }

func (a *archFlag) Type() string { return sizing.ArchX86 + "|" + sizing.ArchARM }

func (a *archFlag) Set(v string) error {
	switch v {
	case sizing.ArchX86, sizing.ArchARM:
		*a = archFlag(v)
		return nil
	}
	return fmt.Errorf("must be one of %s, %s", sizing.ArchX86, sizing.ArchARM)
}

type classFlag string

func (c *classFlag) String() string { return string(*c) }

func (c *classFlag) Type() string {
	return string(sizing.Burstable) + "|" + string(sizing.Dedicated)
}

func (c *classFlag) Set(v string) error {
	switch sizing.Class(v) {
	case sizing.Burstable, sizing.Dedicated:
		*c = classFlag(v)
		return nil
	}
	return fmt.Errorf("must be one of %s, %s", sizing.Burstable, sizing.Dedicated)
}

type shapeFlag string

func (s *shapeFlag) String() string { return string(*s) }

func (s *shapeFlag) Type() string { return "vcpu-memory" }

func (s *shapeFlag) Set(v string) error {
	if slices.Contains(shapeValues(), v) {
		*s = shapeFlag(v)
		return nil
	}
	return fmt.Errorf("must be one of %s", strings.Join(shapeValues(), ", "))
}

type diskFlag int

func (d *diskFlag) String() string {
	if *d == 0 {
		return ""
	}
	return strconv.Itoa(int(*d))
}

func (d *diskFlag) Type() string { return strings.Join(diskValues(), "|") }

func (d *diskFlag) Set(v string) error {
	gb, err := strconv.Atoi(v)
	if err != nil || !slices.Contains(sizing.DiskSizes(), gb) {
		return fmt.Errorf("must be one of %s", strings.Join(diskValues(), ", "))
	}
	*d = diskFlag(gb)
	return nil
}

func shapeValues() []string {
	shapes := sizing.Shapes()
	values := make([]string, len(shapes))
	for i, s := range shapes {
		values[i] = fmt.Sprintf("%dvcpu-%dgb", s.VCPU, s.MemoryGB)
	}
	return values
}

func diskValues() []string {
	sizes := sizing.DiskSizes()
	values := make([]string, len(sizes))
	for i, gb := range sizes {
		values[i] = strconv.Itoa(gb)
	}
	return values
}

func init() {
	createCmd.Flags().Var(&createFlags.arch, "arch", "Processor architecture")
	createCmd.Flags().Var(&createFlags.shape, "shape", "vCPU and memory, one of "+strings.Join(shapeValues(), ", "))
	createCmd.Flags().Var(&createFlags.class, "class", "Instance class")
	createCmd.Flags().Var(&createFlags.disk, "disk", "Root volume size in GB")
}
