package cmd

import (
	"context"
	"fmt"

	"github.com/tanq16/sharingan/internal/awsx"
	u "github.com/tanq16/sharingan/utils"
)

func activeClients(ctx context.Context) *awsx.Clients {
	cfg, err := u.LoadConfig()
	if err != nil {
		u.PrintFatal("Failed to read the sharingan configuration", err)
	}
	profile, region, err := cfg.Active()
	if err != nil {
		u.PrintFatal("No active AWS profile", err)
	}
	clients, err := awsx.New(ctx, profile, region)
	if err != nil {
		u.PrintFatal(fmt.Sprintf("Failed to reach AWS as %s in %s", profile, region), err)
	}
	return clients
}
