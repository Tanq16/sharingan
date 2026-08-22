package awsx

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestTagSpecs(t *testing.T) {
	tests := []struct {
		name     string
		machine  string
		wantTags map[string]string
	}{
		{
			name:     "no name",
			machine:  "",
			wantTags: map[string]string{TagKey: TagValue},
		},
		{
			name:     "named",
			machine:  "serko",
			wantTags: map[string]string{TagKey: TagValue, "Name": "serko"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := TagSpecs(ec2types.ResourceTypeInstance, tt.machine)
			if spec.ResourceType != ec2types.ResourceTypeInstance {
				t.Errorf("ResourceType = %v, want instance", spec.ResourceType)
			}
			got := map[string]string{}
			for _, tag := range spec.Tags {
				got[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
			}
			if len(got) != len(tt.wantTags) {
				t.Fatalf("tags = %v, want %v", got, tt.wantTags)
			}
			for key, want := range tt.wantTags {
				if got[key] != want {
					t.Errorf("tag %s = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}

func TestSingle(t *testing.T) {
	tests := []struct {
		name    string
		found   []string
		want    string
		wantErr bool
	}{
		{name: "no match", found: nil, want: ""},
		{name: "one match", found: []string{"vpc-1"}, want: "vpc-1"},
		{name: "two matches", found: []string{"vpc-1", "vpc-2"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := single("vpc", tt.found)
			if (err != nil) != tt.wantErr {
				t.Fatalf("single() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				for _, id := range tt.found {
					if !strings.Contains(err.Error(), id) {
						t.Errorf("single() err = %q, want it to name %s", err, id)
					}
				}
				return
			}
			if got != tt.want {
				t.Errorf("single() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRootVolumeID(t *testing.T) {
	tests := []struct {
		name string
		inst ec2types.Instance
		want string
	}{
		{
			name: "root device among several",
			inst: ec2types.Instance{
				RootDeviceName: aws.String("/dev/sda1"),
				BlockDeviceMappings: []ec2types.InstanceBlockDeviceMapping{
					{DeviceName: aws.String("/dev/sdb"), Ebs: &ec2types.EbsInstanceBlockDevice{VolumeId: aws.String("vol-data")}},
					{DeviceName: aws.String("/dev/sda1"), Ebs: &ec2types.EbsInstanceBlockDevice{VolumeId: aws.String("vol-root")}},
				},
			},
			want: "vol-root",
		},
		{
			name: "root device name unknown falls back to the first volume",
			inst: ec2types.Instance{
				BlockDeviceMappings: []ec2types.InstanceBlockDeviceMapping{
					{DeviceName: aws.String("/dev/sda1"), Ebs: &ec2types.EbsInstanceBlockDevice{VolumeId: aws.String("vol-root")}},
				},
			},
			want: "vol-root",
		},
		{
			name: "no ebs mappings",
			inst: ec2types.Instance{RootDeviceName: aws.String("/dev/sda1")},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rootVolumeID(tt.inst); got != tt.want {
				t.Errorf("rootVolumeID() = %q, want %q", got, tt.want)
			}
		})
	}
}
