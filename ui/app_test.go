package ui

import (
	"testing"

	"github.com/tkozakas/agent-monitor/client"
)

func TestPruneIdle(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []*treeNode
		wantIDs   []string
		wantCount int
	}{
		{
			name: "all busy nodes are kept",
			nodes: []*treeNode{
				{session: client.Session{ID: "s1"}, status: statusBusy},
				{session: client.Session{ID: "s2"}, status: statusBusy},
			},
			wantCount: 2,
			wantIDs:   []string{"s1", "s2"},
		},
		{
			name: "all idle nodes are removed",
			nodes: []*treeNode{
				{session: client.Session{ID: "s1"}, status: statusIdle},
				{session: client.Session{ID: "s2"}, status: statusIdle},
			},
			wantCount: 0,
			wantIDs:   []string{},
		},
		{
			name: "idle parent with busy child is kept",
			nodes: []*treeNode{
				{
					session: client.Session{ID: "parent"},
					status:  statusIdle,
					children: []*treeNode{
						{session: client.Session{ID: "child"}, status: statusBusy},
					},
				},
			},
			wantCount: 1,
			wantIDs:   []string{"parent", "child"},
		},
		{
			name: "idle parent with idle child is removed",
			nodes: []*treeNode{
				{
					session: client.Session{ID: "parent"},
					status:  statusIdle,
					children: []*treeNode{
						{session: client.Session{ID: "child"}, status: statusIdle},
					},
				},
			},
			wantCount: 0,
			wantIDs:   []string{},
		},
		{
			name: "busy grandchild keeps entire chain",
			nodes: []*treeNode{
				{
					session: client.Session{ID: "grandparent"},
					status:  statusIdle,
					children: []*treeNode{
						{
							session: client.Session{ID: "parent"},
							status:  statusIdle,
							children: []*treeNode{
								{session: client.Session{ID: "grandchild"}, status: statusBusy},
							},
						},
					},
				},
			},
			wantCount: 1,
			wantIDs:   []string{"grandparent", "parent", "grandchild"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pruneIdle(tt.nodes)
			if len(got) != tt.wantCount {
				t.Errorf("pruneIdle() returned %d nodes, want %d", len(got), tt.wantCount)
			}
			ids := flattenTree(got)
			wantSet := make(map[string]bool, len(tt.wantIDs))
			for _, id := range tt.wantIDs {
				wantSet[id] = true
			}
			for _, id := range ids {
				if !wantSet[id] {
					t.Errorf("unexpected id %q in result", id)
				}
			}
		})
	}
}

func TestPruneIdleNestedChildrenPruned(t *testing.T) {
	nodes := []*treeNode{
		{
			session: client.Session{ID: "parent"},
			status:  statusIdle,
			children: []*treeNode{
				{session: client.Session{ID: "busy-child"}, status: statusBusy},
				{session: client.Session{ID: "idle-child"}, status: statusIdle},
			},
		},
	}

	got := pruneIdle(nodes)
	if len(got) != 1 {
		t.Fatalf("expected 1 root, got %d", len(got))
	}
	if len(got[0].children) != 1 {
		t.Fatalf("expected 1 surviving child, got %d", len(got[0].children))
	}
	if got[0].children[0].session.ID != "busy-child" {
		t.Errorf("expected busy-child to survive, got %s", got[0].children[0].session.ID)
	}
}

func TestCycleFocus(t *testing.T) {
	m := New()
	m.panes = []Pane{
		{sessionID: "a"},
		{sessionID: "b"},
	}
	m.focusedPane = focusSidebar // -1

	// Forward: sidebar -> pane 0 -> pane 1 -> sidebar
	m = m.cycleFocus(1)
	if m.focusedPane != 0 {
		t.Errorf("expected pane 0, got %d", m.focusedPane)
	}
	m = m.cycleFocus(1)
	if m.focusedPane != 1 {
		t.Errorf("expected pane 1, got %d", m.focusedPane)
	}
	m = m.cycleFocus(1)
	if m.focusedPane != focusSidebar {
		t.Errorf("expected sidebar, got %d", m.focusedPane)
	}

	// Backward
	m = m.cycleFocus(-1)
	if m.focusedPane != 1 {
		t.Errorf("expected pane 1, got %d", m.focusedPane)
	}
}

func TestMergeRefresh(t *testing.T) {
	m := New()
	m.allSessions = []sessionEntry{
		{session: client.Session{ID: "old1"}, clientIdx: 0},
		{session: client.Session{ID: "old2"}, clientIdx: 1},
	}

	msg := refreshMsg{
		clientIdx: 0,
		sessions:  []client.Session{{ID: "new1"}},
		statuses:  map[string]client.SessionStatus{"new1": {Type: "busy"}},
	}

	m = m.mergeRefresh(msg)
	if len(m.allSessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(m.allSessions))
	}

	ids := map[string]bool{}
	for _, e := range m.allSessions {
		ids[e.session.ID] = true
	}
	if !ids["new1"] {
		t.Error("expected new1 to be present")
	}
	if !ids["old2"] {
		t.Error("expected old2 to be preserved")
	}
	if ids["old1"] {
		t.Error("expected old1 to be replaced")
	}
}

func TestExtractAgentName(t *testing.T) {
	msgs := []client.MessageWithParts{
		{Info: client.Message{Role: "user"}},
		{Info: client.Message{Role: "assistant", Agent: "coder"}},
	}
	if got := extractAgentName(msgs); got != "coder" {
		t.Errorf("expected coder, got %s", got)
	}
}

func TestExtractAgentNameEmpty(t *testing.T) {
	msgs := []client.MessageWithParts{
		{Info: client.Message{Role: "user"}},
	}
	if got := extractAgentName(msgs); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestExtractAgentNameNil(t *testing.T) {
	if got := extractAgentName(nil); got != "" {
		t.Errorf("expected empty for nil, got %s", got)
	}
}
