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
