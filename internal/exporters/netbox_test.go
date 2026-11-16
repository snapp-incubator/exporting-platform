package exporters

import "testing"

func TestParseDiskSizeGB(t *testing.T) {
	tests := []struct {
		text   string
		wantGB int
		wantOK bool
	}{
		// Legacy GB names.
		{"ssd sata 960 gb-s2htnx0j401471", 960, true},
		{"ssd 3840gb", 3840, true},
		// TB names written by the netbox sync since Nov 2026.
		{"ssd sas 3.84 tb-s5g0nc0w101755", 3840, true},
		{"ssd sas 7.68 tb-xyz", 7680, true},
		{"ssd sas 1.6 tb-xyz", 1600, true},
		// Capacity followed by the interface speed: the speed must not win.
		{"1.92 tb 6 gb sata ssd", 1920, true},
		{"3.84 tb 12 gb sas ssd", 3840, true},
		{"samsung 3.84 tb 22.5 gb sas ssd", 3840, true},
		{"ssd 960gb sata 6gb/s", 960, true},
		{"ssd 3.84tb sas 12gbps", 3840, true},
		// Nothing that looks like a capacity.
		{"ssd sas", 0, false},
		{"ssd sata 6gb/s", 0, false},
	}
	for _, tt := range tests {
		gb, ok := parseDiskSizeGB(tt.text)
		if gb != tt.wantGB || ok != tt.wantOK {
			t.Errorf("parseDiskSizeGB(%q) = (%d, %v), want (%d, %v)", tt.text, gb, ok, tt.wantGB, tt.wantOK)
		}
	}
}
