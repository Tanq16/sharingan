package inventory

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/config"
	u "github.com/tanq16/sharingan/utils"
)

const (
	unset       = "-"
	displayTime = "06-Jan-02 15:04 MST"
)

type Machine struct {
	Name         string
	InstanceID   string
	InstanceType string
	Arch         string
	VCPU         int
	MemoryGB     int
	DiskGB       int
	PublicIP     string
	State        string
	Created      time.Time
}

func FromState(account, region string) ([]Machine, error) {
	state, err := config.LoadState()
	if err != nil {
		return nil, err
	}
	return fromRegionState(state.LookupRegion(account, region)), nil
}

func FromAPI(ctx context.Context, c *awsx.Clients) ([]Machine, error) {
	instances, err := c.ManagedInstances(ctx)
	if err != nil {
		return nil, err
	}
	shapes, err := instanceShapes(ctx, c, instances)
	if err != nil {
		return nil, err
	}
	return fromInstances(instances, shapes), nil
}

func Table(machines []Machine) ([]string, [][]string) {
	headers := []string{"NAME", "INSTANCE ID", "TYPE", "ARCH", "VCPU", "MEMORY", "DISK", "PUBLIC IP", "STATE", "CREATED"}
	loc := u.LocalLocation()
	rows := make([][]string, 0, len(machines))
	for _, m := range machines {
		rows = append(rows, []string{
			text(m.Name),
			text(m.InstanceID),
			text(m.InstanceType),
			text(m.Arch),
			number(m.VCPU),
			quantity(m.MemoryGB, "GB"),
			quantity(m.DiskGB, "GB"),
			text(m.PublicIP),
			text(m.State),
			timestamp(m.Created, loc),
		})
	}
	return headers, rows
}

type shape struct {
	vcpu     int
	memoryGB int
}

func fromRegionState(rs *config.RegionState) []Machine {
	if rs == nil {
		return nil
	}
	machines := make([]Machine, 0, len(rs.Machines))
	for name, m := range rs.Machines {
		if m == nil {
			continue
		}
		machines = append(machines, Machine{
			Name:         name,
			InstanceID:   m.InstanceID,
			InstanceType: m.InstanceType,
			Arch:         m.Arch,
			VCPU:         m.VCPU,
			MemoryGB:     m.MemoryGB,
			DiskGB:       m.DiskGB,
			PublicIP:     m.PublicIP,
			State:        m.State,
			Created:      m.Created,
		})
	}
	sortMachines(machines)
	return machines
}

func fromInstances(instances []awsx.Instance, shapes map[string]shape) []Machine {
	machines := make([]Machine, 0, len(instances))
	for _, instance := range instances {
		s := shapes[instance.Type]
		machines = append(machines, Machine{
			Name:         instance.Name,
			InstanceID:   instance.ID,
			InstanceType: instance.Type,
			Arch:         instance.Arch,
			VCPU:         s.vcpu,
			MemoryGB:     s.memoryGB,
			DiskGB:       instance.DiskGB,
			PublicIP:     instance.PublicIP,
			State:        instance.State,
			Created:      instance.Launched,
		})
	}
	sortMachines(machines)
	return machines
}

func instanceShapes(ctx context.Context, c *awsx.Clients, instances []awsx.Instance) (map[string]shape, error) {
	var types []ec2types.InstanceType
	for _, instance := range instances {
		t := ec2types.InstanceType(instance.Type)
		if instance.Type != "" && !slices.Contains(types, t) {
			types = append(types, t)
		}
	}
	shapes := make(map[string]shape, len(types))
	if len(types) == 0 {
		return shapes, nil
	}

	pager := ec2.NewDescribeInstanceTypesPaginator(c.EC2, &ec2.DescribeInstanceTypesInput{InstanceTypes: types})
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("ec2:DescribeInstanceTypes: %w", err)
		}
		for _, info := range page.InstanceTypes {
			var s shape
			if info.VCpuInfo != nil {
				s.vcpu = int(aws.ToInt32(info.VCpuInfo.DefaultVCpus))
			}
			if info.MemoryInfo != nil {
				s.memoryGB = int(aws.ToInt64(info.MemoryInfo.SizeInMiB) / 1024)
			}
			shapes[string(info.InstanceType)] = s
		}
	}
	return shapes, nil
}

func sortMachines(machines []Machine) {
	slices.SortFunc(machines, func(a, b Machine) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.InstanceID, b.InstanceID))
	})
}

func text(value string) string {
	if value == "" {
		return unset
	}
	return value
}

func number(value int) string {
	if value == 0 {
		return unset
	}
	return strconv.Itoa(value)
}

func quantity(value int, unit string) string {
	if value == 0 {
		return unset
	}
	return strconv.Itoa(value) + " " + unit
}

func timestamp(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return unset
	}
	return t.In(loc).Format(displayTime)
}
