package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func withHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		contents  string
		writeFile bool
		want      *Config
		wantErr   bool
	}{
		{
			name:      "registered profiles",
			contents:  `{"current":"prod-admin","profiles":{"prod-admin":{"region":"us-east-2"},"dev":{"region":"eu-west-1"}}}`,
			writeFile: true,
			want: &Config{Current: "prod-admin", Profiles: map[string]*Profile{
				"prod-admin": {Region: "us-east-2"},
				"dev":        {Region: "eu-west-1"},
			}},
		},
		{
			name: "no file is an empty registry",
			want: &Config{Profiles: map[string]*Profile{}},
		},
		{
			name:      "empty object",
			contents:  `{}`,
			writeFile: true,
			want:      &Config{Profiles: map[string]*Profile{}},
		},
		{
			name:      "null profiles",
			contents:  `{"current":"dev","profiles":null}`,
			writeFile: true,
			want:      &Config{Current: "dev", Profiles: map[string]*Profile{}},
		},
		{
			name:      "malformed json",
			contents:  `{"current":`,
			writeFile: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withHome(t)
			if tt.writeFile {
				if err := os.MkdirAll(Dir(), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(ConfigPath(), []byte(tt.contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestConfigActive(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantProfile string
		wantRegion  string
		wantErr     bool
		wantHint    bool
	}{
		{
			name:        "current profile with a region",
			config:      Config{Current: "dev", Profiles: map[string]*Profile{"dev": {Region: "us-east-2"}}},
			wantProfile: "dev",
			wantRegion:  "us-east-2",
		},
		{
			name:     "nothing registered",
			config:   Config{Profiles: map[string]*Profile{}},
			wantErr:  true,
			wantHint: true,
		},
		{
			name:     "no current profile",
			config:   Config{Profiles: map[string]*Profile{"dev": {Region: "us-east-2"}}},
			wantErr:  true,
			wantHint: true,
		},
		{
			name:    "current names an unregistered profile",
			config:  Config{Current: "gone", Profiles: map[string]*Profile{"dev": {Region: "us-east-2"}}},
			wantErr: true,
		},
		{
			name:     "current profile has no region",
			config:   Config{Current: "dev", Profiles: map[string]*Profile{"dev": {}}},
			wantErr:  true,
			wantHint: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, region, err := tt.config.Active()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Active() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if tt.wantHint && !strings.Contains(err.Error(), "set-config") {
					t.Errorf("Active() err = %q, want it to name set-config", err)
				}
				return
			}
			if profile != tt.wantProfile || region != tt.wantRegion {
				t.Errorf("Active() = %q, %q, want %q, %q", profile, region, tt.wantProfile, tt.wantRegion)
			}
		})
	}
}

func TestConfigRegisterAndUse(t *testing.T) {
	cfg := &Config{}
	cfg.Register("dev", "us-east-2")
	cfg.Register("prod", "eu-west-1")

	if cfg.Current != "prod" {
		t.Errorf("Current = %q, want the most recently registered profile", cfg.Current)
	}
	if !slices.Equal(cfg.Names(), []string{"dev", "prod"}) {
		t.Errorf("Names() = %v, want [dev prod]", cfg.Names())
	}
	if cfg.Region("dev") != "us-east-2" {
		t.Errorf("Region(dev) = %q, want us-east-2", cfg.Region("dev"))
	}
	if cfg.Region("absent") != "" {
		t.Errorf("Region(absent) = %q, want an empty string", cfg.Region("absent"))
	}

	if err := cfg.Use("dev"); err != nil {
		t.Fatalf("Use(dev) err = %v", err)
	}
	if cfg.Current != "dev" {
		t.Errorf("Current = %q, want dev", cfg.Current)
	}
	if err := cfg.Use("absent"); err == nil {
		t.Error("Use(absent) err = nil, want an unregistered profile to be refused")
	}
	if cfg.Current != "dev" {
		t.Errorf("Current = %q, want a refused switch to leave it alone", cfg.Current)
	}
}

func TestAvailableProfiles(t *testing.T) {
	awsConfig := `[default]
region = us-east-1

[profile dev]
region = us-east-2

  [profile prod-admin]
region = us-west-2

[sso-session corp]
sso_start_url = https://example.com

[services my-services]
ec2 =

# [profile commented-out]
[profile
`
	credentials := `[legacy]
aws_access_key_id = AKIAEXAMPLE

[default]
aws_access_key_id = AKIAEXAMPLE
`

	tests := []struct {
		name        string
		config      string
		credentials string
		want        []string
	}{
		{
			name:        "both files",
			config:      awsConfig,
			credentials: credentials,
			want:        []string{"default", "dev", "legacy", "prod-admin"},
		},
		{
			name:   "config only",
			config: awsConfig,
			want:   []string{"default", "dev", "prod-admin"},
		},
		{
			name:        "credentials only",
			credentials: credentials,
			want:        []string{"default", "legacy"},
		},
		{
			name: "neither file",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := withHome(t)
			if err := os.MkdirAll(filepath.Join(home, ".aws"), 0o700); err != nil {
				t.Fatal(err)
			}
			for name, body := range map[string]string{"config": tt.config, "credentials": tt.credentials} {
				if body == "" {
					continue
				}
				if err := os.WriteFile(filepath.Join(home, ".aws", name), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got, err := AvailableProfiles()
			if err != nil {
				t.Fatalf("AvailableProfiles() err = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("AvailableProfiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimezoneFrom(t *testing.T) {
	tests := []struct {
		name   string
		target string
		link   bool
		want   string
	}{
		{"linux zoneinfo path", "/usr/share/zoneinfo/Asia/Tokyo", true, "Asia/Tokyo"},
		{"macos zoneinfo path", "/var/db/timezone/zoneinfo/America/New_York", true, "America/New_York"},
		{"single segment zone", "/usr/share/zoneinfo/UTC", true, "UTC"},
		{"no zoneinfo segment", "/etc/UTC", true, defaultTimezone},
		{"nothing after the segment", "/usr/share/zoneinfo/", true, defaultTimezone},
		{"absent link", "", false, defaultTimezone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "localtime")
			if tt.link {
				if err := os.Symlink(tt.target, path); err != nil {
					t.Fatalf("symlink: %v", err)
				}
			}
			if got := timezoneFrom(path); got != tt.want {
				t.Errorf("timezoneFrom(-> %q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestTimezoneFromPlainFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "localtime")
	if err := os.WriteFile(path, []byte("tzdata"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := timezoneFrom(path); got != defaultTimezone {
		t.Errorf("timezoneFrom(regular file) = %q, want %q", got, defaultTimezone)
	}
}

func TestLocationFor(t *testing.T) {
	instant := time.Date(2026, 8, 20, 15, 4, 5, 0, time.UTC)
	tests := []struct {
		name       string
		in         string
		wantOffset int
	}{
		{"utc", "UTC", 0},
		{"the fallback zone resolves", defaultTimezone, 0},
		{"an empty name is utc rather than the host zone", "", 0},
		{"a half hour offset zone", "Asia/Kolkata", 5*60*60 + 30*60},
		{"southern hemisphere winter has no dst", "Australia/Sydney", 10 * 60 * 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, got := instant.In(locationFor(tt.in)).Zone(); got != tt.wantOffset {
				t.Errorf("locationFor(%q) offset = %d, want %d", tt.in, got, tt.wantOffset)
			}
		})
	}
}

func TestLocationForUnknownZone(t *testing.T) {
	if got := locationFor("Not/AZone"); got != time.Local {
		t.Errorf("locationFor(unknown) = %v, want the host zone", got)
	}
}
