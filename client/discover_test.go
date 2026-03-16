package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindServerPortFromFile(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(ocDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(ocDir, "server.json"),
		[]byte(`{"port": 4096}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	port, err := FindServerPort("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 4096 {
		t.Errorf("expected port 4096, got %d", port)
	}
}

func TestFindServerPortNotFound(t *testing.T) {
	dir := t.TempDir()

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	_, err := FindServerPort("")
	if err == nil {
		t.Error("expected error when no server file exists")
	}
}

func TestFindAllServerPorts(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(ocDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(ocDir, "server.json"),
		[]byte(`{"port": 4096, "directory": "/home/user/project1"}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(ocDir, "other.json"),
		[]byte(`{"port": 5000, "directory": "/home/user/project2"}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	servers, err := FindAllServerPorts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	ports := map[int]bool{}
	for _, s := range servers {
		ports[s.Port] = true
	}
	if !ports[4096] || !ports[5000] {
		t.Errorf("expected ports 4096 and 5000, got %v", servers)
	}
}

func TestFindAllServerPortsDedup(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(ocDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(ocDir, "a.json"),
		[]byte(`{"port": 4096}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(ocDir, "b.json"),
		[]byte(`{"port": 4096}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	servers, err := FindAllServerPorts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server after dedup, got %d", len(servers))
	}
}
