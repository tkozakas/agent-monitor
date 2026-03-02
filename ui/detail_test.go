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
	result := renderDetail(nil, "", nil, 80)
	if result == "" {
		t.Error("expected non-empty output for nil session")
	}
}

func TestRenderDetailVerySmallWidth(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test session",
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	todos := []client.Todo{
		{ID: "t1", Content: "This is a long todo content that should not cause a panic", Status: "in_progress"},
	}

	// width=5 should not panic (maxLen = 5-10 = -5, guard prevents truncation)
	result := renderDetail(sess, "busy", todos, 5)
	if result == "" {
		t.Error("expected non-empty output for width=5")
	}
	// The todo content should appear untruncated since maxLen <= 3
	if !strings.Contains(result, "This is a long todo") {
		t.Error("expected full todo content when width too small to truncate")
	}
}

func TestRenderDetailEdgeCaseWidth(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test",
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	todos := []client.Todo{
		{ID: "t1", Content: "Short", Status: "completed"},
	}

	// width=12 -> maxLen = 12-10 = 2, which is <= 3 so no truncation
	result := renderDetail(sess, "idle", todos, 12)
	if result == "" {
		t.Error("expected non-empty output for width=12")
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

	result := renderDetail(sess, "busy", nil, 80)
	if !strings.Contains(result, "Changes:") {
		t.Error("expected summary section in output")
	}
}
