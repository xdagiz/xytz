package version

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{a: "1.0.0", b: "1.0.0", expected: 0},
		{a: "1.0.0", b: "2.0.0", expected: -1},
		{a: "2.0.0", b: "1.0.0", expected: 1},
		{a: "1.0.0", b: "1.0.1", expected: -1},
		{a: "1.0.1", b: "1.0.0", expected: 1},
		{a: "v1.0.0", b: "1.0.0", expected: 0},
		{a: "1.0", b: "1.0.1", expected: -1},
		{a: "", b: "1.0.0", expected: -1},
		{a: "1.0.0", b: "", expected: 1},
		{a: "", b: "", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "" {
				tt.name = tt.a + "_vs_" + tt.b
			}
			result := CompareVersions(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
