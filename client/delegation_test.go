package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDelegationFile(t *testing.T) {
	dir := t.TempDir()
	content := `title: Fix auth bug
description: Fixed the authentication middleware
agent: coder
status: complete
---
Result content here`

	path := filepath.Join(dir, "swift-amber-falcon.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, _ := os.Stat(path)
	d, err := parseDelegationFile(path, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if d.ID != "swift-amber-falcon" {
		t.Errorf("expected id swift-amber-falcon, got %s", d.ID)
	}
	if d.Title != "Fix auth bug" {
		t.Errorf("expected title 'Fix auth bug', got %s", d.Title)
	}
	if d.Agent != "coder" {
		t.Errorf("expected agent coder, got %s", d.Agent)
	}
	if d.Status != "complete" {
		t.Errorf("expected status complete, got %s", d.Status)
	}
}

func TestParseDelegationFileMissingStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("title: Test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info, _ := os.Stat(path)
	d, err := parseDelegationFile(path, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != "unknown" {
		t.Errorf("expected default status 'unknown', got %s", d.Status)
	}
}

func TestReadDelegations(t *testing.T) {
	dir := t.TempDir()
	delegDir := filepath.Join(dir, opencodeDir, delegationsDir)
	if err := os.MkdirAll(delegDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "title: Build feature\nagent: coder\nstatus: complete\n"
	if err := os.WriteFile(filepath.Join(delegDir, "task-one.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	c := New("http://localhost:0")
	delegations, err := c.ReadDelegations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(delegations) != 1 {
		t.Fatalf("expected 1 delegation, got %d", len(delegations))
	}
	if delegations[0].ID != "task-one" {
		t.Errorf("expected id task-one, got %s", delegations[0].ID)
	}
	if delegations[0].Title != "Build feature" {
		t.Errorf("expected title 'Build feature', got %s", delegations[0].Title)
	}
	if delegations[0].Agent != "coder" {
		t.Errorf("expected agent coder, got %s", delegations[0].Agent)
	}
}

func TestReadDelegationsEmptyStateDir(t *testing.T) {
	orig := stateDirectory
	stateDirectory = func() string { return "" }
	defer func() { stateDirectory = orig }()

	c := New("http://localhost:0")
	delegations, err := c.ReadDelegations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delegations != nil {
		t.Errorf("expected nil delegations, got %v", delegations)
	}
}

func TestReadDelegationsNoDelegationsDir(t *testing.T) {
	dir := t.TempDir()

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	c := New("http://localhost:0")
	delegations, err := c.ReadDelegations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delegations != nil {
		t.Errorf("expected nil delegations, got %v", delegations)
	}
}
