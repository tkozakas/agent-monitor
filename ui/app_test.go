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
	// Idle parent has one busy child and one idle child.
	// After pruning: parent kept, busy child kept, idle child removed.
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
