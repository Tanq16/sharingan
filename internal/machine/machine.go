package machine

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/config"
)

//go:embed userdata.sh
var bootstrapScript string

const (
	archX86 = "x86_64"
	archARM = "arm64"

	sshUser       = "ubuntu"
	rootDevice    = "/dev/sda1"
	timezoneToken = "__WORKSTATION_TZ__"

	waitTimeout = 10 * time.Minute
)

// The parameter path spells x86 as amd64, while every architecture comparison holds x86_64.
func amiParameter(arch string) (string, error) {
	var slug string
	switch arch {
	case archX86:
		slug = "amd64"
	case archARM:
		slug = archARM
	default:
		return "", fmt.Errorf("unsupported architecture %q, want %s or %s", arch, archX86, archARM)
	}
	return "/aws/service/canonical/ubuntu/server/26.04/stable/current/" + slug + "/hvm/ebs-gp3/ami-id", nil
}

func resolveAMI(ctx context.Context, c *awsx.Clients, arch string) (string, error) {
	name, err := amiParameter(arch)
	if err != nil {
		return "", err
	}
	out, err := c.SSM.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
	if err != nil {
		return "", fmt.Errorf("ssm:GetParameter %s: %w", name, err)
	}
	if out.Parameter == nil || aws.ToString(out.Parameter.Value) == "" {
		return "", fmt.Errorf("ssm:GetParameter %s returned no value", name)
	}
	return aws.ToString(out.Parameter.Value), nil
}

func renderUserData(timezone string) string {
	return strings.ReplaceAll(bootstrapScript, timezoneToken, timezone)
}

func requireInstance(ctx context.Context, c *awsx.Clients, name string) (*awsx.Instance, error) {
	inst, err := c.FindInstance(ctx, name)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, fmt.Errorf("no machine named %q in %s", name, c.Region)
	}
	return inst, nil
}

func publicIP(ctx context.Context, c *awsx.Clients, instanceID string) (string, error) {
	out, err := c.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}})
	if err != nil {
		return "", fmt.Errorf("ec2:DescribeInstances %s: %w", instanceID, err)
	}
	for _, reservation := range out.Reservations {
		for _, inst := range reservation.Instances {
			return aws.ToString(inst.PublicIpAddress), nil
		}
	}
	return "", fmt.Errorf("ec2:DescribeInstances found no instance %s", instanceID)
}

func waitRunning(ctx context.Context, c *awsx.Clients, instanceID string) error {
	input := &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}
	if err := ec2.NewInstanceRunningWaiter(c.EC2).Wait(ctx, input, waitTimeout); err != nil {
		return fmt.Errorf("waiting for %s to start: %w", instanceID, err)
	}
	return nil
}

func waitStopped(ctx context.Context, c *awsx.Clients, instanceID string) error {
	input := &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}
	if err := ec2.NewInstanceStoppedWaiter(c.EC2).Wait(ctx, input, waitTimeout); err != nil {
		return fmt.Errorf("waiting for %s to stop: %w", instanceID, err)
	}
	return nil
}

func waitTerminated(ctx context.Context, c *awsx.Clients, instanceID string) error {
	input := &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}
	if err := ec2.NewInstanceTerminatedWaiter(c.EC2).Wait(ctx, input, waitTimeout); err != nil {
		return fmt.Errorf("waiting for %s to terminate: %w", instanceID, err)
	}
	return nil
}

// The entry is created when absent, so a deleted state.json heals on the next operation.
func updateMachine(c *awsx.Clients, name string, apply func(*config.Machine)) (*config.Machine, error) {
	state, err := config.LoadState()
	if err != nil {
		return nil, err
	}
	region := state.Region(c.Account, c.Region)
	entry, ok := region.Machines[name]
	if !ok {
		entry = &config.Machine{}
		region.Machines[name] = entry
	}
	apply(entry)
	if err := state.Save(); err != nil {
		return nil, err
	}
	return entry, nil
}

// Whatever only the cache knows, such as the shape a type was chosen for, survives the refresh.
func observe(entry *config.Machine, inst *awsx.Instance) {
	entry.InstanceID = inst.ID
	entry.InstanceType = inst.Type
	entry.Arch = inst.Arch
	entry.State = inst.State
	entry.PublicIP = inst.PublicIP
	if inst.DiskGB > 0 {
		entry.DiskGB = inst.DiskGB
	}
	if entry.Created.IsZero() {
		entry.Created = inst.Launched
	}
}
