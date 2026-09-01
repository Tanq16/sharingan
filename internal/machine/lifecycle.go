package machine

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/tanq16/sharingan/internal/awsx"
)

type StartConfig struct {
	Name string
}

func Start(ctx context.Context, c *awsx.Clients, cfg StartConfig) (string, error) {
	inst, err := requireInstance(ctx, c, cfg.Name)
	if err != nil {
		return "", err
	}
	if err := startInstance(ctx, c, inst.ID); err != nil {
		return "", err
	}
	if err := waitRunning(ctx, c, inst.ID); err != nil {
		return "", err
	}
	// The address is assigned during the transition, so it is only readable once the instance runs.
	return publicIP(ctx, c, inst.ID)
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
	return waitStopped(ctx, c, inst.ID)
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
	return waitTerminated(ctx, c, inst.ID)
}

func startInstance(ctx context.Context, c *awsx.Clients, instanceID string) error {
	if _, err := c.EC2.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		return fmt.Errorf("ec2:StartInstances %s: %w", instanceID, err)
	}
	return nil
}
