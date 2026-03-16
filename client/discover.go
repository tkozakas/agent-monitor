package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	envXDGStateHome = "XDG_STATE_HOME"
	opencodeDir     = "opencode"
	serverFileName  = "server.json"
	serverFileGlob  = "*.json"
	darwinStateDir  = "Library/Application Support"
	linuxStateDir   = ".local/share"
)

var stateDirectory = defaultStateDirectory

func FindServerPort(projectDir string) (int, error) {
	stateDir := stateDirectory()
	if stateDir == "" {
		return 0, fmt.Errorf("cannot determine state directory")
	}

	lockFile := filepath.Join(stateDir, opencodeDir, serverFileName)
	data, err := os.ReadFile(lockFile)
	if err != nil {
		return tryAlternateDiscovery(stateDir, projectDir)
	}

	var info struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return 0, fmt.Errorf("parse %s: %w", serverFileName, err)
	}

	if info.Port > 0 {
		return info.Port, nil
	}
	return 0, fmt.Errorf("no port found in %s", serverFileName)
}

func tryAlternateDiscovery(stateDir, projectDir string) (int, error) {
	pattern := filepath.Join(stateDir, opencodeDir, serverFileGlob)
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return 0, fmt.Errorf("no opencode server found in %s", stateDir)
	}

	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var info struct {
			Port      int    `json:"port"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		if info.Port > 0 && (projectDir == "" || info.Directory == projectDir) {
			return info.Port, nil
		}
	}

	return 0, fmt.Errorf("no matching opencode server for directory %s", projectDir)
}

// ServerInfo holds discovered server port and associated directory.
type ServerInfo struct {
	Port      int
	Directory string
}

// FindAllServerPorts scans the opencode state directory and returns all
// running server ports with their project directories.
func FindAllServerPorts() ([]ServerInfo, error) {
	stateDir := stateDirectory()
	if stateDir == "" {
		return nil, fmt.Errorf("cannot determine state directory")
	}

	pattern := filepath.Join(stateDir, opencodeDir, serverFileGlob)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	var servers []ServerInfo
	seen := make(map[int]bool)
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var info struct {
			Port      int    `json:"port"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal(data, &info); err != nil || info.Port <= 0 {
			continue
		}
		if seen[info.Port] {
			continue
		}
		seen[info.Port] = true
		servers = append(servers, ServerInfo{Port: info.Port, Directory: info.Directory})
	}

	return servers, nil
}

func defaultStateDirectory() string {
	if dir := os.Getenv(envXDGStateHome); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, darwinStateDir)
	default:
		return filepath.Join(home, linuxStateDir)
	}
}
