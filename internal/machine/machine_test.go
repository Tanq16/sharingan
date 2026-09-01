package machine

import (
	"slices"
	"strings"
	"testing"

	"github.com/tanq16/sharingan/internal/awsx"
)

func TestAMIParameter(t *testing.T) {
	tests := []struct {
		name    string
		arch    string
		want    string
		wantErr bool
	}{
		{"x86 spells amd64 in the path", "x86_64", "/aws/service/canonical/ubuntu/server/26.04/stable/current/amd64/hvm/ebs-gp3/ami-id", false},
		{"arm keeps its spelling", "arm64", "/aws/service/canonical/ubuntu/server/26.04/stable/current/arm64/hvm/ebs-gp3/ami-id", false},
		{"the path spelling is not an architecture", "amd64", "", true},
		{"empty", "", "", true},
		{"i386 is never supported", "i386", "", true},
		{"case matters", "ARM64", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := amiParameter(tt.arch)
			if (err != nil) != tt.wantErr {
				t.Fatalf("amiParameter(%q) err = %v, wantErr %v", tt.arch, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("amiParameter(%q) = %q, want %q", tt.arch, got, tt.want)
			}
		})
	}
}

func TestRenderUserData(t *testing.T) {
	rendered := renderUserData("Asia/Kolkata")

	if !strings.Contains(rendered, "timedatectl set-timezone Asia/Kolkata || true") {
		t.Error("rendered script does not set the timezone it was given")
	}
	if strings.Contains(rendered, timezoneToken) {
		t.Errorf("placeholder %s survived rendering", timezoneToken)
	}
	if !strings.Contains(rendered, "touch /home/ubuntu/.sharingan_bootstrap_done") {
		t.Error("rendered script does not write the sharingan sentinel")
	}
}

func TestSSHArgs(t *testing.T) {
	base := []string{
		"ssh",
		"-i", "/keys/id_ed25519",
		"-o", "UserKnownHostsFile=/keys/known_hosts",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "HostKeyAlias=sharingan-123456789012-us-east-2-serko",
		"ubuntu@3.129.210.133",
	}
	tests := []struct {
		name        string
		passthrough []string
		want        []string
	}{
		{"empty passthrough", []string{}, base},
		{"port forward", []string{"-L", "8080:localhost:80"}, append(slices.Clone(base), "-L", "8080:localhost:80")},
		{"remote command", []string{probeCommand}, append(slices.Clone(base), probeCommand)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sshArgs("sharingan-123456789012-us-east-2-serko", "3.129.210.133", "/keys/id_ed25519", "/keys/known_hosts", tt.passthrough)
			if !slices.Equal(got, tt.want) {
				t.Errorf("sshArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateModify(t *testing.T) {
	current := &awsx.Instance{ID: "i-0123456789abcdef0", Type: "m7g.large", Arch: "arm64", State: "running"}

	tests := []struct {
		name    string
		cfg     ModifyConfig
		wantErr bool
	}{
		{
			"same arch, bigger shape",
			ModifyConfig{Name: "serko", InstanceType: "m7g.xlarge", VCPU: 4, MemoryGB: 16, Arches: []string{"arm64"}},
			false,
		},
		{
			"arch change is refused",
			ModifyConfig{Name: "serko", InstanceType: "m7i.xlarge", VCPU: 4, MemoryGB: 16, Arches: []string{"x86_64"}},
			true,
		},
		{
			"multi-arch type keeps the machine arch",
			ModifyConfig{Name: "serko", InstanceType: "m7g.2xlarge", VCPU: 8, MemoryGB: 32, Arches: []string{"x86_64", "arm64"}},
			false,
		},
		{
			"no architectures reported",
			ModifyConfig{Name: "serko", InstanceType: "m7g.xlarge", VCPU: 4, MemoryGB: 16, Arches: nil},
			true,
		},
		{
			"same type is not a change",
			ModifyConfig{Name: "serko", InstanceType: "m7g.large", VCPU: 2, MemoryGB: 8, Arches: []string{"arm64"}},
			true,
		},
		{
			"missing type",
			ModifyConfig{Name: "serko", VCPU: 4, MemoryGB: 16, Arches: []string{"arm64"}},
			true,
		},
		{
			"zero vcpu",
			ModifyConfig{Name: "serko", InstanceType: "m7g.xlarge", MemoryGB: 16, Arches: []string{"arm64"}},
			true,
		},
		{
			"negative memory",
			ModifyConfig{Name: "serko", InstanceType: "m7g.xlarge", VCPU: 4, MemoryGB: -16, Arches: []string{"arm64"}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModify(tt.cfg, current)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateModify(%+v) err = %v, wantErr %v", tt.cfg, err, tt.wantErr)
			}
		})
	}
}

func TestCreateConfigValidate(t *testing.T) {
	valid := CreateConfig{Name: "serko", InstanceType: "m7g.xlarge", Arch: "arm64", VCPU: 4, MemoryGB: 16, DiskGB: 120}

	tests := []struct {
		name    string
		mutate  func(*CreateConfig)
		wantErr bool
	}{
		{"valid", func(*CreateConfig) {}, false},
		{"no name", func(c *CreateConfig) { c.Name = "" }, true},
		{"no type", func(c *CreateConfig) { c.InstanceType = "" }, true},
		{"zero disk", func(c *CreateConfig) { c.DiskGB = 0 }, true},
		{"negative disk", func(c *CreateConfig) { c.DiskGB = -50 }, true},
		{"ssm arch spelling", func(c *CreateConfig) { c.Arch = "amd64" }, true},
		{"unknown arch", func(c *CreateConfig) { c.Arch = "" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)
			if err := cfg.validate(); (err != nil) != tt.wantErr {
				t.Fatalf("CreateConfig.validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
