package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
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

type ServerInfo struct {
	Port      int
	Directory string
}

func FindServerPort(projectDir string) (int, error) {
	servers, err := FindAllServerPorts()
	if err != nil {
		return 0, err
	}
	if len(servers) == 0 {
		return 0, fmt.Errorf("no opencode servers found")
	}
	for _, s := range servers {
		if projectDir == "" || s.Directory == projectDir {
			return s.Port, nil
		}
	}
	return servers[0].Port, nil
}

func FindAllServerPorts() ([]ServerInfo, error) {
	var servers []ServerInfo
	seen := make(map[int]bool)

	if s, err := findFromStateDir(); err == nil {
		for _, srv := range s {
			if !seen[srv.Port] {
				seen[srv.Port] = true
				servers = append(servers, srv)
			}
		}
	}

	if procs, err := findFromProcesses(); err == nil {
		for _, srv := range procs {
			if !seen[srv.Port] {
				seen[srv.Port] = true
				servers = append(servers, srv)
			}
		}
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no opencode servers found")
	}
	return servers, nil
}

func findFromStateDir() ([]ServerInfo, error) {
	stateDir := stateDirectory()
	if stateDir == "" {
		return nil, fmt.Errorf("cannot determine state directory")
	}

	pattern := filepath.Join(stateDir, opencodeDir, serverFileGlob)
	matches, _ := filepath.Glob(pattern)

	var servers []ServerInfo
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var info struct {
			Port      int    `json:"port"`
			Directory string `json:"directory"`
		}
		if json.Unmarshal(data, &info) != nil || info.Port <= 0 {
			continue
		}
		servers = append(servers, ServerInfo{Port: info.Port, Directory: info.Directory})
	}
	return servers, nil
}

var portFlag = regexp.MustCompile(`--port\s+(\d+)`)

func findFromProcesses() ([]ServerInfo, error) {
	out, err := exec.Command("ps", "ax", "-o", "args=").Output()
	if err != nil {
		return nil, err
	}

	var servers []ServerInfo
	seen := make(map[int]bool)
	probe := &http.Client{Timeout: 500 * time.Millisecond}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "opencode") {
			continue
		}
		if strings.Contains(line, "attach") || strings.Contains(line, "grep") {
			continue
		}

		if m := portFlag.FindStringSubmatch(line); len(m) == 2 {
			port, _ := strconv.Atoi(m[1])
			if port > 0 && !seen[port] {
				seen[port] = true
				dir := probeDirectory(probe, port)
				servers = append(servers, ServerInfo{Port: port, Directory: dir})
			}
		}
	}

	return servers, nil
}

func probeDirectory(c *http.Client, port int) string {
	resp, err := c.Get(fmt.Sprintf("http://localhost:%d/session", port))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var sessions []struct {
		Directory string `json:"directory"`
	}
	if json.NewDecoder(resp.Body).Decode(&sessions) == nil && len(sessions) > 0 {
		return sessions[0].Directory
	}
	return ""
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
