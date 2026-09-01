package inventory

import (
	"slices"
	"testing"
	"time"

	"github.com/tanq16/sharingan/internal/awsx"
)

func TestFromInstances(t *testing.T) {
	launched := time.Date(2026, 8, 20, 15, 4, 5, 0, time.UTC)
	shapes := map[string]shape{"m7g.xlarge": {vcpu: 4, memoryGB: 16}}

	tests := []struct {
		name      string
		instances []awsx.Instance
		want      []Machine
	}{
		{name: "no instances", instances: nil, want: []Machine{}},
		{
			name: "shape comes from the api lookup, not from state",
			instances: []awsx.Instance{{
				ID: "i-1", Name: "serko", Type: "m7g.xlarge", Arch: "arm64",
				State: "running", PublicIP: "3.129.210.133", DiskGB: 120, Launched: launched,
			}},
			want: []Machine{{
				Profile: "work", Region: "us-east-2",
				Name: "serko", InstanceID: "i-1", InstanceType: "m7g.xlarge", Arch: "arm64",
				VCPU: 4, MemoryGB: 16, DiskGB: 120, PublicIP: "3.129.210.133",
				State: "running", Created: launched,
			}},
		},
		{
			name:      "unknown instance type leaves the shape unset",
			instances: []awsx.Instance{{ID: "i-1", Name: "serko", Type: "m9z.nano", State: "running"}},
			want: []Machine{{
				Profile: "work", Region: "us-east-2",
				Name: "serko", InstanceID: "i-1", InstanceType: "m9z.nano", State: "running",
			}},
		},
		{
			name: "instances with no name tag sort ahead and keep their id",
			instances: []awsx.Instance{
				{ID: "i-2", Name: "serko", State: "stopped"},
				{ID: "i-1", State: "running"},
			},
			want: []Machine{
				{Profile: "work", Region: "us-east-2", InstanceID: "i-1", State: "running"},
				{Profile: "work", Region: "us-east-2", Name: "serko", InstanceID: "i-2", State: "stopped"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fromInstances(tt.instances, shapes, "work", "us-east-2")
			if !slices.Equal(got, tt.want) {
				t.Errorf("fromInstances() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTable(t *testing.T) {
	headers, rows := Table(nil)
	if len(rows) != 0 {
		t.Errorf("Table(nil) rows = %d, want 0", len(rows))
	}

	machines := []Machine{
		{
			Name: "serko", InstanceID: "i-1", InstanceType: "m7g.xlarge", Arch: "arm64",
			VCPU: 4, MemoryGB: 16, DiskGB: 120, PublicIP: "3.129.210.133", State: "running",
			Created: time.Date(2026, 8, 20, 15, 4, 5, 0, time.UTC),
		},
		{InstanceID: "i-2", State: "stopped"},
	}
	headers, rows = Table(machines)
	for i, row := range rows {
		if len(row) != len(headers) {
			t.Fatalf("row %d has %d cells, want %d", i, len(row), len(headers))
		}
	}

	want := []string{"serko", "i-1", "m7g.xlarge", "arm64", "4", "16 GB", "120 GB", "3.129.210.133", "running"}
	if !slices.Equal(rows[0][:len(want)], want) {
		t.Errorf("row 0 = %v, want %v", rows[0][:len(want)], want)
	}
	wantUnset := []string{unset, "i-2", unset, unset, unset, unset, unset, unset, "stopped", unset}
	if !slices.Equal(rows[1], wantUnset) {
		t.Errorf("row 1 = %v, want %v", rows[1], wantUnset)
	}
}

func TestTimestamp(t *testing.T) {
	instant := time.Date(2026, 8, 20, 15, 4, 5, 0, time.UTC)
	tests := []struct {
		name string
		in   time.Time
		loc  *time.Location
		want string
	}{
		{"zero time", time.Time{}, time.UTC, unset},
		{"utc", instant, time.UTC, "26-Aug-20 15:04 UTC"},
		{"named zone shifts the date back", instant, time.FixedZone("EDT", -4*60*60), "26-Aug-20 11:04 EDT"},
		{"named zone rolls over midnight", instant, time.FixedZone("JST", 9*60*60), "26-Aug-21 00:04 JST"},
		{"unnamed zone falls back to the offset", instant, time.FixedZone("", 5*60*60+30*60), "26-Aug-20 20:34 +0530"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timestamp(tt.in, tt.loc); got != tt.want {
				t.Errorf("timestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}
