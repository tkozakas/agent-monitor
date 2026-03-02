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
