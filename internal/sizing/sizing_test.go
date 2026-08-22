package sizing

import (
	"slices"
	"testing"
)

type fakePrices map[string]float64

func (f fakePrices) ComputeHourly(instanceType string) (float64, bool) {
	rate, ok := f[instanceType]
	return rate, ok
}

// Every one of these is still reported as current-generation, and price alone would pick them.
func TestAllowlistExcludesOldFamilies(t *testing.T) {
	families := allowedFamilies()
	for _, family := range []string{"t2", "m5a", "c6a", "m6g", "r6g"} {
		t.Run(family, func(t *testing.T) {
			if slices.Contains(families, family) {
				t.Errorf("allowlist contains %s", family)
			}
		})
	}
}

func testResolver() *Resolver {
	return &Resolver{
		prices: fakePrices{
			"c7a.medium": 0.0513,
			"c8a.medium": 0.0600,
			"c7g.medium": 0.0361,
			"t3.medium":  0.0417,
			"t3a.medium": 0.0376,
			"t3.large":   0.0835,
			"m7g.large":  0.0816,
			"m8g.large":  0.0500,
			"m5a.large":  0.0500,
		},
		types: []typeInfo{
			{name: "c7a.medium", vcpu: 1, memoryMiB: 2048, arches: []string{ArchX86}},
			{name: "c7g.medium", vcpu: 1, memoryMiB: 2048, arches: []string{ArchARM}},
			{name: "c8a.medium", vcpu: 1, memoryMiB: 2048, arches: []string{ArchX86}},
			{name: "m5a.large", vcpu: 2, memoryMiB: 8192, arches: []string{ArchX86}},
			{name: "m7g.large", vcpu: 2, memoryMiB: 8192, arches: []string{ArchARM}},
			{name: "m8g.large", vcpu: 2, memoryMiB: 8192, arches: []string{ArchX86}},
			{name: "r7g.large", vcpu: 2, memoryMiB: 16384, arches: []string{ArchARM}},
			{name: "t3.large", vcpu: 2, memoryMiB: 8192, arches: []string{"i386", ArchX86}},
			{name: "t3.medium", vcpu: 2, memoryMiB: 4096, arches: []string{"i386", ArchX86}},
			{name: "t3a.medium", vcpu: 2, memoryMiB: 4096, arches: []string{ArchX86}},
		},
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		shape   Shape
		arch    string
		class   Class
		want    string
		wantErr bool
	}{
		{"cheapest of two matching dedicated types", Shape{1, 2}, ArchX86, Dedicated, "c7a.medium", false},
		{"cheapest of two matching burstable types", Shape{2, 4}, ArchX86, Burstable, "t3a.medium", false},
		{"arm dedicated at one vcpu", Shape{1, 2}, ArchARM, Dedicated, "c7g.medium", false},
		{"no burstable type at one vcpu", Shape{1, 2}, ArchX86, Burstable, "", false},
		{"no burstable type at one vcpu on arm", Shape{1, 2}, ArchARM, Burstable, "", false},
		{"arch is a membership test, not the first element", Shape{2, 8}, ArchX86, Burstable, "t3.large", false},
		{"allowlisted type whose arch list lacks the request is rejected", Shape{2, 8}, ArchARM, Dedicated, "m7g.large", false},
		{"a matching type outside the allowlist is rejected", Shape{2, 8}, ArchX86, Dedicated, "", false},
		{"an unpriced candidate is not resolved", Shape{2, 16}, ArchARM, Dedicated, "", false},
		{"a shape no type carries", Shape{16, 128}, ArchARM, Dedicated, "", false},
		{"memory must match exactly", Shape{2, 9}, ArchARM, Dedicated, "", false},
		{"unknown arch", Shape{2, 4}, "riscv64", Dedicated, "", true},
		{"unknown class", Shape{2, 4}, ArchX86, Class("spot"), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testResolver().Resolve(tt.shape, tt.arch, tt.class)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Resolve(%v, %s, %s) err = %v, wantErr %v", tt.shape, tt.arch, tt.class, err, tt.wantErr)
			}
			if tt.want == "" {
				if got != nil {
					t.Fatalf("Resolve(%v, %s, %s) = %+v, want nil", tt.shape, tt.arch, tt.class, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("Resolve(%v, %s, %s) = nil, want %s", tt.shape, tt.arch, tt.class, tt.want)
			}
			if got.InstanceType != tt.want {
				t.Errorf("InstanceType = %s, want %s", got.InstanceType, tt.want)
			}
			if got.VCPU != tt.shape.VCPU || got.MemoryGB != tt.shape.MemoryGB {
				t.Errorf("shape = %d/%d, want %d/%d", got.VCPU, got.MemoryGB, tt.shape.VCPU, tt.shape.MemoryGB)
			}
			if got.Arch != tt.arch || got.Class != tt.class {
				t.Errorf("arch/class = %s/%s, want %s/%s", got.Arch, got.Class, tt.arch, tt.class)
			}
		})
	}
}

func TestDetailsFromCache(t *testing.T) {
	vcpu, memoryGB, arches, err := testResolver().Details(t.Context(), "t3.medium")
	if err != nil {
		t.Fatalf("Details() err = %v", err)
	}
	if vcpu != 2 || memoryGB != 4 {
		t.Errorf("Details() = %d vcpu, %d GB, want 2 vcpu, 4 GB", vcpu, memoryGB)
	}
	if !slices.Contains(arches, ArchX86) || slices.Contains(arches, ArchARM) {
		t.Errorf("arches = %v, want x86_64 without arm64", arches)
	}
}
