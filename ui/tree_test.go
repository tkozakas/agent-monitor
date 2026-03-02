package ui

import (
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

	tree := buildTree(sessions, statuses)
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

	tree := buildTree(sessions, statuses)
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

	tree := buildTree(sessions, nil)
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

	tree := buildTree(sessions, nil)
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
	tree := buildTree(sessions, nil)
	if tree[0].status != "idle" {
		t.Errorf("expected default status idle, got %s", tree[0].status)
	}
}
