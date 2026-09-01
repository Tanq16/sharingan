package cmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/inventory"
	u "github.com/tanq16/sharingan/utils"
)

const showWorkers = 8

var showFlags struct {
	all bool
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "List the machines AWS reports for the active profile",
	Long:  "List the machines AWS reports right now for the active profile, or with --all for every registered profile.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if showFlags.all {
			showEveryProfile(cmd.Context())
			return
		}
		clients := activeClients(cmd.Context())

		u.PrintRunning(fmt.Sprintf("Querying %s", clients.Region))
		machines, err := inventory.FromAPI(cmd.Context(), clients)
		u.ClearLines(1)
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

func showEveryProfile(ctx context.Context) {
	cfg, err := u.LoadConfig()
	if err != nil {
		u.PrintFatal("Failed to read the sharingan configuration", err)
	}
	names := cfg.Names()
	if len(names) == 0 {
		u.PrintFatal("No AWS profile is registered, run `sharingan set-config` to add one", nil)
	}

	type result struct {
		machines []inventory.Machine
		err      error
	}
	results := make([]result, len(names))
	sem := make(chan struct{}, showWorkers)
	var wg sync.WaitGroup

	u.PrintRunning(fmt.Sprintf("Querying %d profiles", len(names)))
	for i, name := range names {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			results[i].machines, results[i].err = profileMachines(ctx, name, cfg.Region(name))
		})
	}
	wg.Wait()
	u.ClearLines(1)

	var machines []inventory.Machine
	var failed int
	for i, r := range results {
		if r.err != nil {
			failed++
			u.PrintWarn(fmt.Sprintf("Skipped profile %s in %s", names[i], cfg.Region(names[i])), r.err)
			continue
		}
		machines = append(machines, r.machines...)
	}
	if failed == len(names) {
		u.PrintFatal("Failed to query every registered profile", nil)
	}
	if len(machines) == 0 {
		u.PrintInfo("No machines in any registered profile")
		return
	}
	inventory.Sort(machines)
	u.PrintTable(inventory.ScopedTable(machines))
}

func profileMachines(ctx context.Context, profile, region string) ([]inventory.Machine, error) {
	clients, err := awsx.New(ctx, profile, region)
	if err != nil {
		return nil, err
	}
	return inventory.FromAPI(ctx, clients)
}

func init() {
	showCmd.Flags().BoolVar(&showFlags.all, "all", false, "Query every registered profile instead of the active one")
}
