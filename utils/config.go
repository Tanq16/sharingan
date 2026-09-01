package utils

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const setConfigHint = "run `sharingan set-config` to register an AWS profile and region"

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "sharingan")
}

func ConfigPath() string { return filepath.Join(Dir(), "config.json") }

func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o700)
}

type Profile struct {
	Region string `json:"region"`
}

type Config struct {
	Current  string              `json:"current"`
	Profiles map[string]*Profile `json:"profiles"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Profiles: map[string]*Profile{}}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s is not valid json: %w", ConfigPath(), err)
	}
	if c.Profiles == nil {
		c.Profiles = map[string]*Profile{}
	}
	return &c, nil
}

func (c *Config) Save() error {
	if err := EnsureDir(); err != nil {
		return err
	}
	if c.Profiles == nil {
		c.Profiles = map[string]*Profile{}
	}
	data, err := json.Marshal(c, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), append(data, '\n'), 0o600)
}

func (c *Config) Names() []string {
	return slices.Sorted(maps.Keys(c.Profiles))
}

func (c *Config) Region(profile string) string {
	if p := c.Profiles[profile]; p != nil {
		return p.Region
	}
	return ""
}

func (c *Config) Register(profile, region string) {
	if c.Profiles == nil {
		c.Profiles = map[string]*Profile{}
	}
	c.Profiles[profile] = &Profile{Region: region}
	c.Current = profile
}

func (c *Config) Use(profile string) error {
	if _, ok := c.Profiles[profile]; !ok {
		return fmt.Errorf("%s is not registered, the registered profiles are %s", profile, strings.Join(c.Names(), ", "))
	}
	c.Current = profile
	return nil
}

func (c *Config) Active() (string, string, error) {
	if len(c.Profiles) == 0 {
		return "", "", fmt.Errorf("no profile is registered in %s: %s", ConfigPath(), setConfigHint)
	}
	if c.Current == "" {
		return "", "", fmt.Errorf("%s names no current profile: %s", ConfigPath(), setConfigHint)
	}
	profile, ok := c.Profiles[c.Current]
	if !ok {
		return "", "", fmt.Errorf("current profile %s is not registered in %s: run `sharingan use` to pick one", c.Current, ConfigPath())
	}
	if profile.Region == "" {
		return "", "", fmt.Errorf("profile %s has no region in %s: %s", c.Current, ConfigPath(), setConfigHint)
	}
	return c.Current, profile.Region, nil
}

func AvailableProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sources := []struct {
		path      string
		namedOnly bool
	}{
		{filepath.Join(home, ".aws", "config"), true},
		{filepath.Join(home, ".aws", "credentials"), false},
	}

	var names []string
	for _, src := range sources {
		data, err := os.ReadFile(src.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, name := range parseProfiles(string(data), src.namedOnly) {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

// namedOnly is the ~/.aws/config spelling: every section but [default] is [profile x], and [sso-session x] is no profile.
func parseProfiles(data string, namedOnly bool) []string {
	var names []string
	for line := range strings.SplitSeq(data, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		section, _, closed := strings.Cut(strings.TrimPrefix(line, "["), "]")
		if !closed {
			continue
		}
		fields := strings.Fields(section)
		switch {
		case len(fields) == 1 && (!namedOnly || fields[0] == "default"):
			names = append(names, fields[0])
		case len(fields) == 2 && namedOnly && fields[0] == "profile":
			names = append(names, fields[1])
		}
	}
	return names
}
