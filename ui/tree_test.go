package ui

import (
	"strings"
	"testing"

	"github.com/tkozakas/agent-monitor/client"
)

func TestBuildTreeRoots(t *testing.T) {
	sessions := []client.Session{
		{ID: "s1", Title: "root1"},
		{ID: "s2", Title: "root2"},
	}
	statuses := map[string]client.SessionStatus{
		"s1": {Type: "busy"},
		"s2": {Type: "idle"},
	}

	tree := buildTree(sessions, statuses, nil)
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}
}

func TestBuildTreeParentChild(t *testing.T) {
	parentID := "p1"
	sessions := []client.Session{
		{ID: "p1", Title: "parent"},
		{ID: "c1", Title: "child", ParentID: &parentID},
	}
	statuses := map[string]client.SessionStatus{
		"p1": {Type: "busy"},
		"c1": {Type: "idle"},
	}

	tree := buildTree(sessions, statuses, nil)
	if len(tree) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree))
	}
	if len(tree[0].children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree[0].children))
	}
	if tree[0].children[0].session.ID != "c1" {
		t.Errorf("expected child id c1, got %s", tree[0].children[0].session.ID)
	}
}

func TestBuildTreeOrphanChild(t *testing.T) {
	missingParent := "missing"
	sessions := []client.Session{
		{ID: "c1", Title: "orphan", ParentID: &missingParent},
	}

	tree := buildTree(sessions, nil, nil)
	if len(tree) != 1 {
		t.Fatalf("expected orphan promoted to root, got %d roots", len(tree))
	}
}

func TestFlattenTree(t *testing.T) {
	parentID := "p1"
	sessions := []client.Session{
		{ID: "p1", Title: "parent"},
		{ID: "c1", Title: "child1", ParentID: &parentID},
		{ID: "c2", Title: "child2", ParentID: &parentID},
	}

	tree := buildTree(sessions, nil, nil)
	ids := flattenTree(tree)
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d", len(ids))
	}
	if ids[0] != "p1" {
		t.Errorf("expected first id p1, got %s", ids[0])
	}
}

func TestBuildTreeStatusDefault(t *testing.T) {
	sessions := []client.Session{{ID: "s1", Title: "test"}}
	tree := buildTree(sessions, nil, nil)
	if tree[0].status != "idle" {
		t.Errorf("expected default status idle, got %s", tree[0].status)
	}
}

func TestBuildTreeAgentNames(t *testing.T) {
	parentID := "p1"
	sessions := []client.Session{
		{ID: "p1", Title: "session-title"},
		{ID: "c1", Title: "child-title", ParentID: &parentID},
	}
	agents := map[string]string{
		"p1": "build",
		"c1": "coder",
	}

	tree := buildTree(sessions, nil, agents)
	if tree[0].agent != "build" {
		t.Errorf("expected root agent build, got %s", tree[0].agent)
	}
	if tree[0].children[0].agent != "coder" {
		t.Errorf("expected child agent coder, got %s", tree[0].children[0].agent)
	}
}

func TestBuildTreeAgentNameFallback(t *testing.T) {
	sessions := []client.Session{{ID: "s1", Title: "my-session"}}
	tree := buildTree(sessions, nil, nil)
	if tree[0].agent != "" {
		t.Errorf("expected empty agent without agentNames, got %s", tree[0].agent)
	}
}

func TestRenderTreeShowsAgentName(t *testing.T) {
	parentID := "p1"
	sessions := []client.Session{
		{ID: "p1", Title: "session-title"},
		{ID: "c1", Title: "child-title", ParentID: &parentID},
		{ID: "c2", Title: "child-title-2", ParentID: &parentID},
	}
	agents := map[string]string{
		"p1": "build",
		"c1": "coder",
		"c2": "researcher",
	}
	statuses := map[string]client.SessionStatus{
		"p1": {Type: "busy"},
		"c1": {Type: "busy"},
		"c2": {Type: "busy"},
	}

	tree := buildTree(sessions, statuses, agents)
	output := renderTree(tree, "p1", 60)

	if !strings.Contains(output, "build") {
		t.Error("expected tree to contain agent name 'build'")
	}
	if !strings.Contains(output, "coder") {
		t.Error("expected tree to contain agent name 'coder'")
	}
	if !strings.Contains(output, "researcher") {
		t.Error("expected tree to contain agent name 'researcher'")
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

func TestNodeLabel(t *testing.T) {
	tests := []struct {
		name     string
		node     *treeNode
		expected string
	}{
		{
			name:     "agent name takes priority",
			node:     &treeNode{agent: "build", session: client.Session{ID: "abc12345", Title: "my-title"}},
			expected: "build",
		},
		{
			name:     "falls back to title",
			node:     &treeNode{session: client.Session{ID: "abc12345", Title: "my-title"}},
			expected: "my-title",
		},
		{
			name:     "falls back to ID prefix",
			node:     &treeNode{session: client.Session{ID: "abc12345xyz"}},
			expected: "abc12345",
		},
		{
			name:     "truncates long names",
			node:     &treeNode{agent: "this-is-a-very-long-agent-name-that-exceeds-thirty"},
			expected: "this-is-a-very-long-agent-n...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nodeLabel(tt.node)
			if got != tt.expected {
				t.Errorf("nodeLabel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGraphCellWidth(t *testing.T) {
	node := &treeNode{agent: "build", status: "busy", session: client.Session{ID: "s1"}}
	got := graphCellWidth(node)
	expected := 2 + len("build") + 1 + len("busy") // "● build busy" = 2+5+1+4 = 12
	if got != expected {
		t.Errorf("graphCellWidth() = %d, want %d", got, expected)
	}
}

func TestGraphSubtreeWidth(t *testing.T) {
	t.Run("leaf", func(t *testing.T) {
		node := &treeNode{agent: "build", status: "busy", session: client.Session{ID: "s1"}}
		got := graphSubtreeWidth(node)
		expected := graphCellWidth(node)
		if got != expected {
			t.Errorf("graphSubtreeWidth(leaf) = %d, want %d", got, expected)
		}
	})

	t.Run("with children", func(t *testing.T) {
		child1 := &treeNode{agent: "coder", status: "busy", session: client.Session{ID: "c1"}}
		child2 := &treeNode{agent: "researcher", status: "idle", session: client.Session{ID: "c2"}}
		parent := &treeNode{
			agent:    "build",
			status:   "busy",
			session:  client.Session{ID: "p1"},
			children: []*treeNode{child1, child2},
		}
		got := graphSubtreeWidth(parent)
		childrenW := graphCellWidth(child1) + graphNodeGap + graphCellWidth(child2)
		parentW := graphCellWidth(parent)
		expected := childrenW
		if parentW > expected {
			expected = parentW
		}
		if got != expected {
			t.Errorf("graphSubtreeWidth(parent) = %d, want %d", got, expected)
		}
	})
}

func TestRenderGraphTreeSingle(t *testing.T) {
	node := &treeNode{agent: "build", status: "busy", session: client.Session{ID: "s1"}}
	output := renderGraphTree([]*treeNode{node}, "s1", 80)
	if !strings.Contains(output, "build") {
		t.Error("expected output to contain 'build'")
	}
	if !strings.Contains(output, "busy") {
		t.Error("expected output to contain 'busy'")
	}
	if strings.Contains(output, "┌") || strings.Contains(output, "┴") {
		t.Error("single root should not have connectors")
	}
}

func TestRenderGraphTreeWithChildren(t *testing.T) {
	parentID := "p1"
	sessions := []client.Session{
		{ID: "p1", Title: "session"},
		{ID: "c1", Title: "child1", ParentID: &parentID},
		{ID: "c2", Title: "child2", ParentID: &parentID},
		{ID: "c3", Title: "child3", ParentID: &parentID},
	}
	agents := map[string]string{
		"p1": "build",
		"c1": "coder",
		"c2": "researcher",
		"c3": "reviewer",
	}
	statuses := map[string]client.SessionStatus{
		"p1": {Type: "busy"},
		"c1": {Type: "busy"},
		"c2": {Type: "idle"},
		"c3": {Type: "busy"},
	}

	tree := buildTree(sessions, statuses, agents)
	output := renderGraphTree(tree, "p1", 100)

	for _, name := range []string{"build", "coder", "researcher", "reviewer"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}
	if !strings.Contains(output, "┌") {
		t.Error("expected output to contain connector '┌'")
	}
	if !strings.Contains(output, "┴") {
		t.Error("expected output to contain connector '┴'")
	}
	if !strings.Contains(output, "┐") {
		t.Error("expected output to contain connector '┐'")
	}
	if !strings.Contains(output, "┬") {
		t.Error("expected output to contain connector '┬' for intermediate children")
	}
}

func TestRenderGraphConnectorsSingleChild(t *testing.T) {
	child := &layoutNode{
		node:    &treeNode{agent: "coder", status: "busy", session: client.Session{ID: "c1"}},
		centerX: 20,
	}
	parent := &layoutNode{
		node:     &treeNode{agent: "build", status: "busy", session: client.Session{ID: "p1"}},
		centerX:  20,
		children: []*layoutNode{child},
	}

	lines := renderGraphConnectors([]*layoutNode{parent}, 60)
	if len(lines) != 2 {
		t.Fatalf("expected 2 connector lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "│") {
		t.Error("expected line 1 to contain '│'")
	}
	if !strings.Contains(lines[1], "│") {
		t.Error("expected line 2 to contain '│'")
	}
	if strings.Contains(lines[1], "┌") || strings.Contains(lines[1], "┐") {
		t.Error("single child should not have horizontal connectors")
	}
}

func TestBuildGraphLayout(t *testing.T) {
	child1 := &treeNode{agent: "coder", status: "busy", session: client.Session{ID: "c1"}}
	child2 := &treeNode{agent: "researcher", status: "idle", session: client.Session{ID: "c2"}}
	root := &treeNode{
		agent:    "build",
		status:   "busy",
		session:  client.Session{ID: "p1"},
		children: []*treeNode{child1, child2},
	}

	ln := buildGraphLayout(root, 0)

	if ln.totalWidth <= 0 {
		t.Errorf("expected positive totalWidth, got %d", ln.totalWidth)
	}
	if len(ln.children) != 2 {
		t.Fatalf("expected 2 layout children, got %d", len(ln.children))
	}
	if ln.children[0].centerX >= ln.children[1].centerX {
		t.Errorf("expected first child left of second: %d >= %d", ln.children[0].centerX, ln.children[1].centerX)
	}
	if ln.centerX < ln.children[0].centerX || ln.centerX > ln.children[1].centerX {
		t.Errorf("expected parent center (%d) between children (%d, %d)", ln.centerX, ln.children[0].centerX, ln.children[1].centerX)
	}
}

func TestRenderGraphTreeNoRoots(t *testing.T) {
	output := renderGraphTree(nil, "", 80)
	if !strings.Contains(output, "No agents") {
		t.Error("expected 'No agents' for empty tree")
	}
}

func TestRenderGraphTreeDeepNesting(t *testing.T) {
	p1 := "p1"
	c1 := "c1"
	sessions := []client.Session{
		{ID: "p1", Title: "root"},
		{ID: "c1", Title: "mid", ParentID: &p1},
		{ID: "g1", Title: "leaf", ParentID: &c1},
	}
	agents := map[string]string{"p1": "build", "c1": "coder", "g1": "tester"}
	statuses := map[string]client.SessionStatus{
		"p1": {Type: "busy"},
		"c1": {Type: "busy"},
		"g1": {Type: "idle"},
	}

	tree := buildTree(sessions, statuses, agents)
	output := renderGraphTree(tree, "p1", 80)

	for _, name := range []string{"build", "coder", "tester"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}
}
