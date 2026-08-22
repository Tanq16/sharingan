package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	u "github.com/tanq16/sharingan/utils"
)

func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoadStateMissingFile(t *testing.T) {
	withHome(t)

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() err = %v, want a missing file to be an empty state", err)
	}
	if s.Version != 1 {
		t.Errorf("Version = %d, want 1", s.Version)
	}
	if len(s.Accounts) != 0 {
		t.Errorf("Accounts = %v, want empty", s.Accounts)
	}
}

func TestStateRoundTrip(t *testing.T) {
	withHome(t)

	created := time.Date(2026, 8, 20, 15, 4, 5, 0, time.UTC)
	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() err = %v", err)
	}
	region := s.Region("123456789012", "us-east-2")
	region.VPCID = "vpc-1"
	region.SubnetID = "subnet-1"
	region.IGWID = "igw-1"
	region.RouteTableID = "rtb-1"
	region.SecurityGroupID = "sg-1"
	region.KeyPairName = "sharingan-a7f3c1e9"
	region.Machines["serko"] = &Machine{
		InstanceID:   "i-1",
		InstanceType: "m7g.xlarge",
		Arch:         "arm64",
		VCPU:         4,
		MemoryGB:     16,
		DiskGB:       120,
		PublicIP:     "3.129.210.133",
		State:        "running",
		Created:      created,
	}
	s.Region("111111111111", "us-west-2").VPCID = "vpc-2"

	if err := s.Save(); err != nil {
		t.Fatalf("Save() err = %v", err)
	}

	info, err := os.Stat(StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("state.json mode = %v, want 0600", perm)
	}

	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() err = %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
	if !reflect.DeepEqual(loaded.Accounts, s.Accounts) {
		t.Fatalf("Accounts = %+v, want %+v", loaded.Accounts, s.Accounts)
	}

	machine := loaded.LookupRegion("123456789012", "us-east-2").Machines["serko"]
	if machine.InstanceType != "m7g.xlarge" || machine.DiskGB != 120 {
		t.Errorf("machine = %+v, want m7g.xlarge with a 120GB disk", machine)
	}
	if !machine.Created.Equal(created) {
		t.Errorf("Created = %v, want %v", machine.Created, created)
	}
	if loaded.LookupRegion("111111111111", "us-west-2").VPCID != "vpc-2" {
		t.Error("second account did not survive the round trip")
	}
}

func TestStateRegionLookupAndDelete(t *testing.T) {
	s := &State{}

	if got := s.LookupRegion("123456789012", "us-east-2"); got != nil {
		t.Errorf("LookupRegion on an empty state = %+v, want nil", got)
	}

	region := s.Region("123456789012", "us-east-2")
	region.VPCID = "vpc-1"
	if s.Region("123456789012", "us-east-2") != region {
		t.Error("Region returned a different entry on the second call")
	}
	if got := s.LookupRegion("123456789012", "us-east-2"); got != region {
		t.Errorf("LookupRegion = %+v, want the created entry", got)
	}
	if got := s.LookupRegion("123456789012", "eu-west-1"); got != nil {
		t.Errorf("LookupRegion for another region = %+v, want nil", got)
	}

	s.Region("123456789012", "eu-west-1").VPCID = "vpc-2"
	s.DeleteRegion("123456789012", "eu-west-1")
	if _, ok := s.Accounts["123456789012"]; !ok {
		t.Error("deleting one region dropped an account that still has another")
	}

	s.DeleteRegion("123456789012", "us-east-2")
	if _, ok := s.Accounts["123456789012"]; ok {
		t.Error("deleting the last region left an empty account behind")
	}
	s.DeleteRegion("does-not-exist", "us-east-2")
}

func TestStateLoadMachinesNull(t *testing.T) {
	withHome(t)
	if err := os.MkdirAll(u.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"version":1,"accounts":{"123456789012":{"us-east-2":{"vpc_id":"vpc-1","machines":null}}}}`
	if err := os.WriteFile(StatePath(), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() err = %v", err)
	}
	s.LookupRegion("123456789012", "us-east-2").Machines["serko"] = &Machine{InstanceID: "i-1"}
	s.Region("123456789012", "us-east-2").Machines["kabuto"] = &Machine{InstanceID: "i-2"}
	if got := len(s.Region("123456789012", "us-east-2").Machines); got != 2 {
		t.Errorf("machines = %d, want 2", got)
	}
}

func TestEnsureKeyPair(t *testing.T) {
	withHome(t)

	authorized, err := EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() err = %v", err)
	}
	if !strings.HasPrefix(string(authorized), "ssh-ed25519 ") {
		t.Errorf("public half = %q, want an ssh-ed25519 authorized_keys line", authorized)
	}

	info, err := os.Stat(KeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("private key mode = %v, want 0600", perm)
	}

	private, err := os.ReadFile(KeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(private), "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Errorf("private key is not in the OpenSSH format, which is the only one ssh -i reads:\n%s", private)
	}

	again, err := EnsureKeyPair()
	if err != nil {
		t.Fatalf("second EnsureKeyPair() err = %v", err)
	}
	if string(again) != string(authorized) {
		t.Error("EnsureKeyPair() replaced an existing key, which locks the user out of every machine using it")
	}

	if err := os.Remove(PubKeyPath()); err != nil {
		t.Fatal(err)
	}
	derived, err := EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() after removing the public half err = %v", err)
	}
	if string(derived) != string(authorized) {
		t.Error("a missing public half was regenerated instead of derived from the existing private key")
	}
	stored, err := os.ReadFile(PubKeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(authorized) {
		t.Errorf("%s = %q, want %q", PubKeyPath(), stored, authorized)
	}
}
