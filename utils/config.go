package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const setConfigHint = "run `sharingan set-config` to choose an AWS profile and region"

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "sharingan")
}

func ConfigPath() string { return filepath.Join(Dir(), "config.json") }

func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o700)
}

type Config struct {
	Profile string `json:"profile"`
	Region  string `json:"region"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no configuration at %s: %s", ConfigPath(), setConfigHint)
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s is not valid json: %w", ConfigPath(), err)
	}
	var missing []string
	if c.Profile == "" {
		missing = append(missing, "profile")
	}
	if c.Region == "" {
		missing = append(missing, "region")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%s is missing %s: %s", ConfigPath(), strings.Join(missing, " and "), setConfigHint)
	}
	return &c, nil
}

func (c *Config) Save() error {
	if err := EnsureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), append(data, '\n'), 0o600)
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
