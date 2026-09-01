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
	all      bool
	profiles bool
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "List the machines AWS reports for the active profile",
	Long:  "List the machines AWS reports right now for the active profile, with --all for every registered profile or --profiles for the registry itself.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case showFlags.profiles:
			showRegistry(cmd.Context())
		case showFlags.all:
			showEveryProfile(cmd.Context())
		default:
			showActiveProfile(cmd.Context())
		}
	},
}

func showActiveProfile(ctx context.Context) {
	clients := activeClients(ctx)

	u.PrintRunning(fmt.Sprintf("Querying %s", clients.Region))
	machines, err := inventory.FromAPI(ctx, clients)
	u.ClearLines(1)
	if err != nil {
		u.PrintFatal("Failed to read the machine inventory", err)
	}

	if len(machines) == 0 {
		u.PrintInfo(fmt.Sprintf("No machines in account %s region %s", clients.Account, clients.Region))
		return
	}
	u.PrintTable(inventory.Table(machines))
}

func showEveryProfile(ctx context.Context) {
	cfg, names := registeredProfiles()
	results := make([]profileInventory, len(names))

	u.PrintRunning(fmt.Sprintf("Querying %d profiles", len(names)))
	inParallel(len(names), func(i int) {
		results[i] = queryProfile(ctx, names[i], cfg.Region(names[i]))
	})
	u.ClearLines(1)

	var failed int
	for _, r := range results {
		if r.err != nil {
			failed++
			u.PrintWarn(fmt.Sprintf("Skipped profile %s in %s", r.profile, r.region), r.err)
		}
	}
	if failed == len(names) {
		u.PrintFatal("Failed to query every registered profile", nil)
	}

	for _, r := range results {
		if r.err != nil {
			continue
		}
		if len(r.machines) == 0 {
			u.PrintInfo(fmt.Sprintf("No machines in profile %s, account %s, region %s", r.profile, r.account, r.region))
			continue
		}
		u.PrintInfo(fmt.Sprintf("Profile %s, account %s, region %s", r.profile, r.account, r.region))
		u.PrintTable(inventory.Table(r.machines))
	}
}

func showRegistry(ctx context.Context) {
	cfg, names := registeredProfiles()
	accounts := make([]string, len(names))
	errs := make([]error, len(names))

	u.PrintRunning(fmt.Sprintf("Querying %d profiles", len(names)))
	inParallel(len(names), func(i int) {
		clients, err := awsx.New(ctx, names[i], cfg.Region(names[i]))
		if err != nil {
			errs[i] = err
			return
		}
		accounts[i] = clients.Account
	})
	u.ClearLines(1)

	rows := make([][]string, 0, len(names))
	for i, name := range names {
		account := accounts[i]
		if errs[i] != nil {
			u.PrintWarn(fmt.Sprintf("Failed to reach profile %s in %s", name, cfg.Region(name)), errs[i])
			account = "-"
		}
		rows = append(rows, []string{name, account, cfg.Region(name)})
	}
	u.PrintTable([]string{"PROFILE", "ACCOUNT", "REGION"}, rows)
}

type profileInventory struct {
	profile  string
	region   string
	account  string
	machines []inventory.Machine
	err      error
}

func queryProfile(ctx context.Context, profile, region string) profileInventory {
	result := profileInventory{profile: profile, region: region}
	clients, err := awsx.New(ctx, profile, region)
	if err != nil {
		result.err = err
		return result
	}
	result.account = clients.Account
	result.machines, result.err = inventory.FromAPI(ctx, clients)
	return result
}

func registeredProfiles() (*u.Config, []string) {
	cfg, err := u.LoadConfig()
	if err != nil {
		u.PrintFatal("Failed to read the sharingan configuration", err)
	}
	names := cfg.Names()
	if len(names) == 0 {
		u.PrintFatal("No AWS profile is registered, run `sharingan set-config` to add one", nil)
	}
	return cfg, names
}

func inParallel(n int, fn func(int)) {
	sem := make(chan struct{}, showWorkers)
	var wg sync.WaitGroup
	for i := range n {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			fn(i)
		})
	}
	wg.Wait()
}

func init() {
	showCmd.Flags().BoolVar(&showFlags.all, "all", false, "Query every registered profile instead of the active one")
	showCmd.Flags().BoolVar(&showFlags.profiles, "profiles", false, "List the registered profiles with their AWS accounts and regions")
	showCmd.MarkFlagsMutuallyExclusive("all", "profiles")
}
