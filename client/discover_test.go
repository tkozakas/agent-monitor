package client

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestFindFromStateDirWithServerFile(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, "opencode")
	os.MkdirAll(ocDir, 0755)
	os.WriteFile(filepath.Join(ocDir, "server.json"), []byte(`{"port": 4096}`), 0644)

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	servers, err := findFromStateDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 || servers[0].Port != 4096 {
		t.Errorf("expected [{4096}], got %v", servers)
	}
}

func TestFindFromStateDirMultiple(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, "opencode")
	os.MkdirAll(ocDir, 0755)
	os.WriteFile(filepath.Join(ocDir, "a.json"), []byte(`{"port": 4096}`), 0644)
	os.WriteFile(filepath.Join(ocDir, "b.json"), []byte(`{"port": 5000}`), 0644)

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	servers, _ := findFromStateDir()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
}

func TestFindFromStateDirEmpty(t *testing.T) {
	dir := t.TempDir()
	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	servers, _ := findFromStateDir()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

func TestFindFromStateDirInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, "opencode")
	os.MkdirAll(ocDir, 0755)
	os.WriteFile(filepath.Join(ocDir, "bad.json"), []byte(`not json`), 0644)

	orig := stateDirectory
	stateDirectory = func() string { return dir }
	defer func() { stateDirectory = orig }()

	servers, _ := findFromStateDir()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
}

func TestPortFlagRegex(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"opencode --port 33667", 33667},
		{"opencode --port 53335", 53335},
		{"/usr/bin/opencode --port 4096 --other", 4096},
		{"opencode", 0},
		{"opencode attach http://localhost:33667", 0},
	}
	for _, tt := range tests {
		m := portFlag.FindStringSubmatch(tt.line)
		got := 0
		if len(m) == 2 {
			got, _ = strconv.Atoi(m[1])
		}
		if got != tt.want {
			t.Errorf("line %q: expected %d, got %d", tt.line, tt.want, got)
		}
	}
}
