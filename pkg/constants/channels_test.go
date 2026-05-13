package constants

import "testing"

func TestIsInternalChannel(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		want    bool
	}{
		{"cli channel", "cli", true},
		{"system channel", "system", true},
		{"subagent channel", "subagent", true},
		{"telegram channel", "telegram", false},
		{"discord channel", "discord", false},
		{"slack channel", "slack", false},
		{"empty string", "", false},
		{"unknown channel", "matrix", false},
		{"case sensitive - CLI uppercase", "CLI", false},
		{"case sensitive - System uppercase", "System", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInternalChannel(tt.channel)
			if got != tt.want {
				t.Errorf("IsInternalChannel(%q) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}
