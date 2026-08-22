package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	u "github.com/tanq16/sharingan/utils"
)

const stateVersion = 1

func StatePath() string { return filepath.Join(u.Dir(), "state.json") }

type Machine struct {
	InstanceID   string    `json:"instance_id"`
	InstanceType string    `json:"instance_type"`
	Arch         string    `json:"arch"`
	VCPU         int       `json:"vcpu"`
	MemoryGB     int       `json:"memory_gb"`
	DiskGB       int       `json:"disk_gb"`
	PublicIP     string    `json:"public_ip"`
	State        string    `json:"state"`
	Created      time.Time `json:"created"`
}

type RegionState struct {
	VPCID           string              `json:"vpc_id"`
	SubnetID        string              `json:"subnet_id"`
	IGWID           string              `json:"igw_id"`
	RouteTableID    string              `json:"route_table_id"`
	SecurityGroupID string              `json:"security_group_id"`
	KeyPairName     string              `json:"key_pair_name"`
	Machines        map[string]*Machine `json:"machines"`
}

type State struct {
	Version  int                                `json:"version"`
	Accounts map[string]map[string]*RegionState `json:"accounts"`
}

// A missing file is an empty state, because state.json is a cache rebuildable from tags.
func LoadState() (*State, error) {
	data, err := os.ReadFile(StatePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{Version: stateVersion, Accounts: map[string]map[string]*RegionState{}}, nil
		}
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%s is not valid json: %w", StatePath(), err)
	}
	if s.Accounts == nil {
		s.Accounts = map[string]map[string]*RegionState{}
	}
	return &s, nil
}

func (s *State) Save() error {
	if err := u.EnsureDir(); err != nil {
		return err
	}
	s.Version = stateVersion
	if s.Accounts == nil {
		s.Accounts = map[string]map[string]*RegionState{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(), append(data, '\n'), 0o600)
}

// Creates the account and region entry when absent, so the caller can write into it.
func (s *State) Region(account, region string) *RegionState {
	if s.Accounts == nil {
		s.Accounts = map[string]map[string]*RegionState{}
	}
	regions, ok := s.Accounts[account]
	if !ok {
		regions = map[string]*RegionState{}
		s.Accounts[account] = regions
	}
	rs, ok := regions[region]
	if !ok {
		rs = &RegionState{}
		regions[region] = rs
	}
	if rs.Machines == nil {
		rs.Machines = map[string]*Machine{}
	}
	return rs
}

func (s *State) LookupRegion(account, region string) *RegionState {
	rs := s.Accounts[account][region]
	if rs == nil {
		return nil
	}
	if rs.Machines == nil {
		rs.Machines = map[string]*Machine{}
	}
	return rs
}

func (s *State) DeleteRegion(account, region string) {
	regions, ok := s.Accounts[account]
	if !ok {
		return
	}
	delete(regions, region)
	if len(regions) == 0 {
		delete(s.Accounts, account)
	}
}
