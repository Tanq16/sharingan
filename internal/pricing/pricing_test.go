package pricing

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func fixtureEntries(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("testdata/pricelist.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("fixture has %d entries, want 3", len(entries))
	}
	return entries
}

func TestParseEntryFixture(t *testing.T) {
	entries := fixtureEntries(t)
	tests := []struct {
		name         string
		raw          string
		instanceType string
		usd          float64
	}{
		{"compute instance ignores the reserved terms", entries[0], "m7g.large", 0.0816},
		{"gp3 storage carries no instance type", entries[1], "", 0.08},
		{"in-use public ipv4", entries[2], "", 0.005},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEntry(tt.raw)
			if err != nil {
				t.Fatalf("parseEntry() err = %v", err)
			}
			if got.instanceType != tt.instanceType {
				t.Errorf("instanceType = %q, want %q", got.instanceType, tt.instanceType)
			}
			if got.usd != tt.usd {
				t.Errorf("usd = %v, want %v", got.usd, tt.usd)
			}
		})
	}
}

func TestParseEntryErrors(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty string", ""},
		{"malformed json", `{"product":`},
		{"no on-demand term", `{"product":{"attributes":{"instanceType":"m7g.large"}},"terms":{"Reserved":{"a":{"priceDimensions":{"b":{"pricePerUnit":{"USD":"1.0"}}}}}}}`},
		{"on-demand term with no dimensions", `{"product":{"attributes":{"instanceType":"m7g.large"}},"terms":{"OnDemand":{"a":{"priceDimensions":{}}}}}`},
		{"unparseable rate", `{"product":{"attributes":{"instanceType":"m7g.large"}},"terms":{"OnDemand":{"a":{"priceDimensions":{"b":{"pricePerUnit":{"USD":"free"}}}}}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseEntry(tt.raw); err == nil {
				t.Errorf("parseEntry(%q) err = nil, want an error", tt.raw)
			}
		})
	}
}

func TestCostModel(t *testing.T) {
	table := &Table{
		compute:    map[string]float64{"m7g.large": 0.0816},
		gp3GBMonth: 0.08,
		ipv4Hourly: 0.005,
	}
	tests := []struct {
		name    string
		got     float64
		want    float64
		epsilon float64
	}{
		{"running hourly at 120 GB", table.RunningHourly(0.0816, 120), 0.0997506849315069, 1e-12},
		{"running hourly with no disk is compute plus the address", table.RunningHourly(0.0816, 0), 0.0866, 1e-12},
		{"running month at 120 GB", table.RunningMonth(0.0816, 120), 72.818, 1e-9},
		{"stopped month at 120 GB", table.StoppedMonth(120), 9.6, 1e-12},
		{"stopped month at zero disk", table.StoppedMonth(0), 0, 1e-12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if math.Abs(tt.got-tt.want) > tt.epsilon {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}
