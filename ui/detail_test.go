package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/tkozakas/agent-monitor/client"
)

func TestFormatDurationSeconds(t *testing.T) {
	got := formatDuration(45 * time.Second)
	if got != "45s" {
		t.Errorf("expected 45s, got %s", got)
	}
}

func TestFormatDurationMinutes(t *testing.T) {
	got := formatDuration(2*time.Minute + 30*time.Second)
	if got != "2m 30s" {
		t.Errorf("expected 2m 30s, got %s", got)
	}
}

func TestFormatDurationHours(t *testing.T) {
	got := formatDuration(1*time.Hour + 15*time.Minute)
	if got != "1h 15m" {
		t.Errorf("expected 1h 15m, got %s", got)
	}
}

func TestRenderDetailNilSession(t *testing.T) {
	result := renderDetail(nil, "", nil, 80, 40, 0)
	if result == "" {
		t.Error("expected non-empty output for nil session")
	}
}

func TestRenderDetailWithMessages(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test session",
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	msgs := []client.MessageWithParts{
		{
			Info: client.Message{Role: "assistant", Agent: "coder"},
			Parts: []client.Part{
				{Type: "text", Text: "Hello world"},
				{Type: "tool", Tool: "bash", State: &client.ToolState{Status: "completed", Title: "Run tests"}},
			},
		},
	}

	result := renderDetail(sess, "busy", msgs, 80, 40, 0)
	if !strings.Contains(result, "coder") {
		t.Error("expected agent name in output")
	}
	if !strings.Contains(result, "Hello world") {
		t.Error("expected message text in output")
	}
	if !strings.Contains(result, "Run tests") {
		t.Error("expected tool name in output")
	}
}

func TestRenderDetailScrollOffset(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test session",
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	msgs := []client.MessageWithParts{
		{
			Info: client.Message{Role: "assistant", Agent: "build"},
			Parts: []client.Part{
				{Type: "text", Text: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"},
			},
		},
	}

	noScroll := renderDetail(sess, "busy", msgs, 80, 5, 0)
	scrolled := renderDetail(sess, "busy", msgs, 80, 5, 2)

	if noScroll == scrolled {
		t.Error("expected scrolled output to differ from non-scrolled")
	}
}

func TestRenderDetailWithSummary(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test session",
		Summary: &client.SessionSummary{
			Additions: 10,
			Deletions: 5,
			Files:     3,
		},
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	result := renderDetail(sess, "busy", nil, 80, 40, 0)
	if !strings.Contains(result, "3 files") {
		t.Error("expected summary section in output")
	}
}

func TestRenderDetailSmallWidth(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test session",
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	result := renderDetail(sess, "busy", nil, 5, 10, 0)
	if result == "" {
		t.Error("expected non-empty output for small width")
	}
}

func TestClampWidth(t *testing.T) {
	if clampWidth(0) != 1 {
		t.Error("expected clampWidth(0) = 1")
	}
	if clampWidth(-5) != 1 {
		t.Error("expected clampWidth(-5) = 1")
	}
	if clampWidth(50) != 50 {
		t.Error("expected clampWidth(50) = 50")
	}
}

func TestWrapText(t *testing.T) {
	lines := wrapText("hello world", 5)
	if len(lines) < 2 {
		t.Errorf("expected at least 2 lines, got %d", len(lines))
	}
}

func TestToolStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"running", "⟳"},
		{"completed", "✓"},
		{"error", "✗"},
		{"pending", "…"},
	}
	for _, tt := range tests {
		got := toolStatusIcon(tt.status)
		if got != tt.want {
			t.Errorf("toolStatusIcon(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
