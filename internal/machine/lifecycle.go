package machine

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/config"
)

type StartConfig struct {
	Name string
}

func Start(ctx context.Context, c *awsx.Clients, cfg StartConfig) (*config.Machine, error) {
	inst, err := requireInstance(ctx, c, cfg.Name)
	if err != nil {
		return nil, err
	}
	if err := startInstance(ctx, c, inst.ID); err != nil {
		return nil, err
	}
	if err := waitRunning(ctx, c, inst.ID); err != nil {
		return nil, err
	}
	// The address is assigned during the transition, so it is only readable once the instance runs.
	ip, err := publicIP(ctx, c, inst.ID)
	if err != nil {
		return nil, err
	}
	return updateMachine(c, cfg.Name, func(entry *config.Machine) {
		observe(entry, inst)
		entry.PublicIP = ip
		entry.State = string(ec2types.InstanceStateNameRunning)
	})
}

type StopConfig struct {
	Name string
}

func Stop(ctx context.Context, c *awsx.Clients, cfg StopConfig) error {
	inst, err := requireInstance(ctx, c, cfg.Name)
	if err != nil {
		return err
	}
	if _, err := c.EC2.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{inst.ID}}); err != nil {
		return fmt.Errorf("ec2:StopInstances %s: %w", inst.ID, err)
	}
	if err := waitStopped(ctx, c, inst.ID); err != nil {
		return err
	}
	// An auto-assigned public IPv4 address is released on stop, and a new one arrives on the next start.
	_, err = updateMachine(c, cfg.Name, func(entry *config.Machine) {
		observe(entry, inst)
		entry.PublicIP = ""
		entry.State = string(ec2types.InstanceStateNameStopped)
	})
	return err
}

type RemoveConfig struct {
	Name string
}

func Remove(ctx context.Context, c *awsx.Clients, cfg RemoveConfig) error {
	inst, err := requireInstance(ctx, c, cfg.Name)
	if err != nil {
		return err
	}
	if _, err := c.EC2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{inst.ID}}); err != nil {
		return fmt.Errorf("ec2:TerminateInstances %s: %w", inst.ID, err)
	}
	if err := waitTerminated(ctx, c, inst.ID); err != nil {
		return err
	}

	state, err := config.LoadState()
	if err != nil {
		return err
	}
	region := state.LookupRegion(c.Account, c.Region)
	if region == nil {
		return nil
	}
	delete(region.Machines, cfg.Name)
	return state.Save()
}

func startInstance(ctx context.Context, c *awsx.Clients, instanceID string) error {
	if _, err := c.EC2.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		return fmt.Errorf("ec2:StartInstances %s: %w", instanceID, err)
	}
	return nil
}
