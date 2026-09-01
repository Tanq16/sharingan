package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/pricing"
	"github.com/tanq16/sharingan/internal/sizing"
	u "github.com/tanq16/sharingan/utils"
)

var costsCmd = &cobra.Command{
	Use:   "costs",
	Short: "Show monthly cost estimates for every size, arch, and disk combination",
	Long:  "Show monthly cost estimates for every size, arch, and disk combination, priced live for the active region.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		clients := activeClients(ctx)

		u.PrintRunning("fetching on-demand prices")
		prices, err := pricing.Fetch(ctx, clients)
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal("Failed to fetch prices", err)
		}

		u.PrintRunning("resolving instance types")
		resolver, err := sizing.NewResolver(ctx, clients, prices)
		u.ClearLines(1)
		if err != nil {
			u.PrintFatal("Failed to resolve instance types", err)
		}

		disks := sizing.DiskSizes()
		diskHeaders := make([]string, len(disks))
		stopped := make([]string, len(disks))
		for i, gb := range disks {
			diskHeaders[i] = fmt.Sprintf("%d GB", gb)
			stopped[i] = fmt.Sprintf("$%.2f", prices.StoppedMonth(gb))
		}

		runningHeaders := append([]string{"Shape", "Class", "Type", "Compute/hr"}, diskHeaders...)
		for _, arch := range []string{sizing.ArchX86, sizing.ArchARM} {
			u.PrintInfo(fmt.Sprintf("Monthly cost while running on %s in %s, by root volume size", arch, clients.Region))
			u.PrintTable(runningHeaders, runningRows(resolver, prices, arch, disks))
		}
		u.PrintInfo("Monthly cost while stopped, root volume only")
		u.PrintTable(diskHeaders, [][]string{stopped})
		u.PrintInfo("Data transfer is excluded")
	},
}

func runningRows(resolver *sizing.Resolver, prices *pricing.Table, arch string, disks []int) [][]string {
	var rows [][]string
	for _, shape := range sizing.Shapes() {
		for _, class := range []sizing.Class{sizing.Burstable, sizing.Dedicated} {
			resolved, err := resolver.Resolve(shape, arch, class)
			if err != nil {
				u.PrintFatal("Failed to resolve an instance type", err)
			}
			if resolved == nil {
				continue
			}
			row := []string{
				fmt.Sprintf("%d vCPU / %d GB", shape.VCPU, shape.MemoryGB),
				string(class),
				resolved.InstanceType,
				fmt.Sprintf("$%.4f", resolved.HourlyUSD),
			}
			for _, gb := range disks {
				row = append(row, fmt.Sprintf("$%.2f", prices.RunningMonth(resolved.HourlyUSD, gb)))
			}
			rows = append(rows, row)
		}
	}
	return rows
}
