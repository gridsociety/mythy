package configio

import "testing"

func TestKeepConfigChange(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		readOnly bool
		want     bool
	}{
		{"writable Set/ entry is kept", "Set/Base", false, true},
		{"writable Read/ entry is filtered", "Read/Measures", false, false},
		{"READONLY Set/ entry is filtered", "Set/Base", true, false},
		{"READONLY Read/ entry is filtered", "Read/Measures", true, false},
		{"writable Communication/ entry is kept", "Communication/RS485", false, true},
		{"READONLY Communication/ entry is filtered (Eth0_HW_Address case)", "Communication/eth0", true, false},
		{"unknown key (empty path, not RO) is kept", "", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := keepConfigChange(c.path, c.readOnly); got != c.want {
				t.Errorf("keepConfigChange(%q, %v) = %v, want %v", c.path, c.readOnly, got, c.want)
			}
		})
	}
}
