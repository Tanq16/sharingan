package scaffold

import (
	"slices"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/tanq16/sharingan/internal/awsx"
)

func TestBlockingInstances(t *testing.T) {
	tests := []struct {
		name      string
		instances []awsx.Instance
		want      []string
	}{
		{
			name:      "no instances",
			instances: nil,
			want:      nil,
		},
		{
			name:      "terminated does not block",
			instances: []awsx.Instance{{ID: "i-1", Name: "serko", State: "terminated"}},
			want:      nil,
		},
		{
			name:      "running blocks and is named",
			instances: []awsx.Instance{{ID: "i-1", Name: "serko", State: "running"}},
			want:      []string{"serko (i-1, running)"},
		},
		{
			name:      "stopped blocks",
			instances: []awsx.Instance{{ID: "i-1", Name: "serko", State: "stopped"}},
			want:      []string{"serko (i-1, stopped)"},
		},
		{
			name:      "missing name falls back to the id",
			instances: []awsx.Instance{{ID: "i-2", State: "shutting-down"}},
			want:      []string{"i-2 (shutting-down)"},
		},
		{
			name: "mixed states keep order and drop the terminated one",
			instances: []awsx.Instance{
				{ID: "i-1", Name: "serko", State: "running"},
				{ID: "i-2", State: "terminated"},
				{ID: "i-3", Name: "kakashi", State: "stopping"},
			},
			want: []string{"serko (i-1, running)", "kakashi (i-3, stopping)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockingInstances(tt.instances)
			if !slices.Equal(got, tt.want) {
				t.Errorf("blockingInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachedTo(t *testing.T) {
	tests := []struct {
		name     string
		gateways []ec2types.InternetGateway
		want     bool
	}{
		{name: "no gateway", gateways: nil, want: false},
		{
			name:     "gateway with no attachment",
			gateways: []ec2types.InternetGateway{{InternetGatewayId: aws.String("igw-1")}},
			want:     false,
		},
		{
			name: "attached to another vpc",
			gateways: []ec2types.InternetGateway{{Attachments: []ec2types.InternetGatewayAttachment{
				{VpcId: aws.String("vpc-other")},
			}}},
			want: false,
		},
		{
			name: "attachment with no vpc id",
			gateways: []ec2types.InternetGateway{{Attachments: []ec2types.InternetGatewayAttachment{
				{State: ec2types.AttachmentStatusAttached},
			}}},
			want: false,
		},
		{
			name: "attached to the target vpc",
			gateways: []ec2types.InternetGateway{{Attachments: []ec2types.InternetGatewayAttachment{
				{VpcId: aws.String("vpc-other")},
				{VpcId: aws.String("vpc-1")},
			}}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := attachedTo(tt.gateways, "vpc-1"); got != tt.want {
				t.Errorf("attachedTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasDefaultRoute(t *testing.T) {
	tests := []struct {
		name   string
		tables []ec2types.RouteTable
		want   bool
	}{
		{name: "no table", tables: nil, want: false},
		{
			name:   "only the local route",
			tables: []ec2types.RouteTable{{Routes: []ec2types.Route{{DestinationCidrBlock: aws.String("10.20.0.0/16")}}}},
			want:   false,
		},
		{
			name:   "ipv6 route carries no ipv4 destination",
			tables: []ec2types.RouteTable{{Routes: []ec2types.Route{{DestinationIpv6CidrBlock: aws.String("::/0")}}}},
			want:   false,
		},
		{
			name: "default route present",
			tables: []ec2types.RouteTable{{Routes: []ec2types.Route{
				{DestinationCidrBlock: aws.String("10.20.0.0/16")},
				{DestinationCidrBlock: aws.String("0.0.0.0/0")},
			}}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDefaultRoute(tt.tables); got != tt.want {
				t.Errorf("hasDefaultRoute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssociatedWith(t *testing.T) {
	tests := []struct {
		name   string
		tables []ec2types.RouteTable
		want   bool
	}{
		{name: "no table", tables: nil, want: false},
		{
			name: "main association carries no subnet",
			tables: []ec2types.RouteTable{{Associations: []ec2types.RouteTableAssociation{
				{Main: aws.Bool(true), RouteTableAssociationId: aws.String("rtbassoc-1")},
			}}},
			want: false,
		},
		{
			name: "associated with another subnet",
			tables: []ec2types.RouteTable{{Associations: []ec2types.RouteTableAssociation{
				{SubnetId: aws.String("subnet-other")},
			}}},
			want: false,
		},
		{
			name: "associated with the target subnet",
			tables: []ec2types.RouteTable{{Associations: []ec2types.RouteTableAssociation{
				{Main: aws.Bool(true)},
				{SubnetId: aws.String("subnet-1")},
			}}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := associatedWith(tt.tables, "subnet-1"); got != tt.want {
				t.Errorf("associatedWith() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMissingIngress(t *testing.T) {
	rule := func(protocol string, from, to int32, cidr string) ec2types.IpPermission {
		return ec2types.IpPermission{
			IpProtocol: aws.String(protocol),
			FromPort:   aws.Int32(from),
			ToPort:     aws.Int32(to),
			IpRanges:   []ec2types.IpRange{{CidrIp: aws.String(cidr)}},
		}
	}

	tests := []struct {
		name   string
		groups []ec2types.SecurityGroup
		want   []int32
	}{
		{name: "group not described", groups: nil, want: []int32{22, 443}},
		{
			name:   "no rules",
			groups: []ec2types.SecurityGroup{{GroupId: aws.String("sg-1")}},
			want:   []int32{22, 443},
		},
		{
			name:   "both rules present",
			groups: []ec2types.SecurityGroup{{IpPermissions: []ec2types.IpPermission{rule("tcp", 22, 22, anywhere), rule("tcp", 443, 443, anywhere)}}},
			want:   nil,
		},
		{
			name:   "only ssh present",
			groups: []ec2types.SecurityGroup{{IpPermissions: []ec2types.IpPermission{rule("tcp", 22, 22, anywhere)}}},
			want:   []int32{443},
		},
		{
			name:   "right port from the wrong source",
			groups: []ec2types.SecurityGroup{{IpPermissions: []ec2types.IpPermission{rule("tcp", 22, 22, "10.0.0.0/8")}}},
			want:   []int32{22, 443},
		},
		{
			name:   "right port on the wrong protocol",
			groups: []ec2types.SecurityGroup{{IpPermissions: []ec2types.IpPermission{rule("udp", 22, 22, anywhere)}}},
			want:   []int32{22, 443},
		},
		{
			name:   "all protocols carries no port",
			groups: []ec2types.SecurityGroup{{IpPermissions: []ec2types.IpPermission{{IpProtocol: aws.String("-1"), IpRanges: []ec2types.IpRange{{CidrIp: aws.String(anywhere)}}}}}},
			want:   []int32{22, 443},
		},
		{
			name: "ipv6 only rule does not count",
			groups: []ec2types.SecurityGroup{{IpPermissions: []ec2types.IpPermission{{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(443),
				ToPort:     aws.Int32(443),
				Ipv6Ranges: []ec2types.Ipv6Range{{CidrIpv6: aws.String("::/0")}},
			}}}},
			want: []int32{22, 443},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missingIngress(tt.groups, ingressPorts)
			if !slices.Equal(got, tt.want) {
				t.Errorf("missingIngress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubnetAssociations(t *testing.T) {
	tests := []struct {
		name   string
		tables []ec2types.RouteTable
		want   []string
	}{
		{name: "no table", tables: nil, want: nil},
		{
			name: "main association is skipped",
			tables: []ec2types.RouteTable{{Associations: []ec2types.RouteTableAssociation{
				{Main: aws.Bool(true), RouteTableAssociationId: aws.String("rtbassoc-main")},
			}}},
			want: nil,
		},
		{
			name: "association without an id is skipped",
			tables: []ec2types.RouteTable{{Associations: []ec2types.RouteTableAssociation{
				{SubnetId: aws.String("subnet-1")},
			}}},
			want: nil,
		},
		{
			name: "subnet associations are returned",
			tables: []ec2types.RouteTable{{Associations: []ec2types.RouteTableAssociation{
				{Main: aws.Bool(true), RouteTableAssociationId: aws.String("rtbassoc-main")},
				{SubnetId: aws.String("subnet-1"), RouteTableAssociationId: aws.String("rtbassoc-1")},
				{SubnetId: aws.String("subnet-2"), RouteTableAssociationId: aws.String("rtbassoc-2")},
			}}},
			want: []string{"rtbassoc-1", "rtbassoc-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := subnetAssociations(tt.tables)
			if !slices.Equal(got, tt.want) {
				t.Errorf("subnetAssociations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachedVPCs(t *testing.T) {
	tests := []struct {
		name     string
		gateways []ec2types.InternetGateway
		want     []string
	}{
		{name: "no gateway", gateways: nil, want: nil},
		{
			name:     "detached gateway",
			gateways: []ec2types.InternetGateway{{InternetGatewayId: aws.String("igw-1")}},
			want:     nil,
		},
		{
			name: "attachment without a vpc id is skipped",
			gateways: []ec2types.InternetGateway{{Attachments: []ec2types.InternetGatewayAttachment{
				{State: ec2types.AttachmentStatusAttaching},
			}}},
			want: nil,
		},
		{
			name: "every attached vpc is returned",
			gateways: []ec2types.InternetGateway{{Attachments: []ec2types.InternetGatewayAttachment{
				{VpcId: aws.String("vpc-1")},
				{VpcId: aws.String("vpc-2")},
			}}},
			want: []string{"vpc-1", "vpc-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attachedVPCs(tt.gateways)
			if !slices.Equal(got, tt.want) {
				t.Errorf("attachedVPCs() = %v, want %v", got, tt.want)
			}
		})
	}
}
