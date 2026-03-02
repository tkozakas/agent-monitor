package ui

import (
	"testing"
)

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"busy"},
		{"idle"},
		{"error"},
		{"retry"},
		{"complete"},
		{"completed"},
		{"unknown_fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			c := statusColor(tt.status)
			if c == nil {
				t.Errorf("statusColor(%q) returned nil", tt.status)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"busy", "●"},
		{"idle", "○"},
		{"error", "✗"},
		{"retry", "↻"},
		{"complete", "✓"},
		{"completed", "✓"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := statusIcon(tt.status)
			if got != tt.want {
				t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestTodoIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"completed", "✓"},
		{"in_progress", "→"},
		{"cancelled", "✗"},
		{"pending", "○"},
		{"unknown", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := todoIcon(tt.status)
			if got != tt.want {
				t.Errorf("todoIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestTodoStatusColor(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"completed"},
		{"in_progress"},
		{"cancelled"},
		{"pending"},
		{"unknown_fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			c := todoStatusColor(tt.status)
			if c == nil {
				t.Errorf("todoStatusColor(%q) returned nil", tt.status)
			}
		})
	}
}
