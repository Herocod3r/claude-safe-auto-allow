package version

import "testing"

func TestGt(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want bool
	}{
		{"1.1.0", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"0.9.0", "1.0.0", false},
		{"", "1.0.0", false},
		{"1.0.0", "", false},
		{"invalid", "1.0.0", false},
		{"1.0.0", "invalid", false},
	}

	for _, tt := range tests {
		got := Gt(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("Gt(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
