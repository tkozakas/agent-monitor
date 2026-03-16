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
	result := renderDetail(nil, "", nil, 80, 40, 0, false)
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

	result := renderDetail(sess, "busy", msgs, 80, 40, 0, false)
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

	noScroll := renderDetail(sess, "busy", msgs, 80, 5, 0, false)
	scrolled := renderDetail(sess, "busy", msgs, 80, 5, 2, false)

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

	result := renderDetail(sess, "busy", nil, 80, 40, 0, false)
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

	result := renderDetail(sess, "busy", nil, 5, 10, 0, false)
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

func TestAggregateCost(t *testing.T) {
	msgs := []client.MessageWithParts{
		{
			Info: client.Message{
				Cost: 0.25,
				Tokens: &client.Tokens{
					Input:     1000,
					Output:    500,
					Reasoning: 200,
				},
			},
		},
		{
			Info: client.Message{
				Cost: 0.17,
				Tokens: &client.Tokens{
					Input:  2000,
					Output: 300,
				},
			},
		},
	}

	cost, in, out, reasoning, cache := aggregateCost(msgs)
	if cost < 0.41 || cost > 0.43 {
		t.Errorf("expected cost ~0.42, got %f", cost)
	}
	if in != 3000 {
		t.Errorf("expected in 3000, got %d", in)
	}
	if out != 800 {
		t.Errorf("expected out 800, got %d", out)
	}
	if reasoning != 200 {
		t.Errorf("expected reasoning 200, got %d", reasoning)
	}
	if cache != 0 {
		t.Errorf("expected cache 0, got %d", cache)
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0k"},
		{12345, "12.3k"},
		{100000, "100.0k"},
	}
	for _, tt := range tests {
		got := formatTokens(tt.n)
		if got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestRenderSparkline(t *testing.T) {
	result := renderSparkline(45, 15, 40, 20)
	if !strings.Contains(result, "in:") {
		t.Error("expected in: in sparkline")
	}
	if !strings.Contains(result, "out:") {
		t.Error("expected out: in sparkline")
	}
	if !strings.Contains(result, "cache:") {
		t.Error("expected cache: in sparkline")
	}
}

func TestRenderSparklineZero(t *testing.T) {
	result := renderSparkline(0, 0, 0, 20)
	if result != "" {
		t.Error("expected empty sparkline for zero totals")
	}
}

func TestRenderDetailExpandTools(t *testing.T) {
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
				{Type: "tool", Tool: "bash", State: &client.ToolState{
					Status: "completed",
					Title:  "Run tests",
					Output: "all tests passed",
				}},
			},
		},
	}

	collapsed := renderDetail(sess, "busy", msgs, 80, 40, 0, false)
	expanded := renderDetail(sess, "busy", msgs, 80, 40, 0, true)

	if collapsed == expanded {
		t.Error("expected expanded output to differ from collapsed")
	}
	if !strings.Contains(expanded, "all tests passed") {
		t.Error("expected tool output in expanded view")
	}
}

func TestRenderPaneNilSession(t *testing.T) {
	p := newPane("empty", 0)
	result := renderPane(p, nil, "", nil, 60, 20, false, false)
	if !strings.Contains(result, "No session") {
		t.Error("expected 'No session' for nil session pane")
	}
}

func TestRenderPaneFocused(t *testing.T) {
	sess := &client.Session{
		ID:    "abcdef123456",
		Title: "test session",
	}
	sess.Time.Created = time.Now().Add(-1 * time.Minute).UnixMilli()
	sess.Time.Updated = time.Now().UnixMilli()

	p := newPane("abcdef123456", 0)
	focused := renderPane(p, sess, "busy", nil, 60, 20, true, false)
	unfocused := renderPane(p, sess, "busy", nil, 60, 20, false, false)

	if focused == unfocused {
		t.Error("expected focused and unfocused panes to render differently")
	}
}

func TestRenderPaneSmallDimensions(t *testing.T) {
	p := newPane("s1", 0)
	result := renderPane(p, nil, "", nil, 2, 2, false, false)
	if result != "" {
		t.Error("expected empty for very small dimensions")
	}
}
