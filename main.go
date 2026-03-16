package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/tkozakas/agent-monitor/client"
	"github.com/tkozakas/agent-monitor/ui"
)

const (
	envURL  = "OPENCODE_URL"
	envPort = "OPENCODE_PORT"
	envHost = "OPENCODE_HOST"
	envDir  = "OPENCODE_DIR"

	defaultHost   = "localhost"
	defaultScheme = "http"
)

func main() {
	clients := resolveClients()
	if len(clients) == 0 {
		fmt.Fprintln(os.Stderr, "no opencode servers found; set OPENCODE_URL or start opencode first")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(clients...))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func resolveClients() []client.OpenCodeClient {
	// Explicit URL takes priority — single server mode
	if url := os.Getenv(envURL); url != "" {
		return []client.OpenCodeClient{client.New(url)}
	}

	host := os.Getenv(envHost)
	if host == "" {
		host = defaultHost
	}

	// Explicit port — single server mode
	if portStr := os.Getenv(envPort); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil && port > 0 {
			url := fmt.Sprintf("%s://%s:%d", defaultScheme, host, port)
			return []client.OpenCodeClient{client.New(url)}
		}
	}

	// Auto-discover ALL running opencode servers
	servers, err := client.FindAllServerPorts()
	if err == nil && len(servers) > 0 {
		clients := make([]client.OpenCodeClient, 0, len(servers))
		for _, s := range servers {
			url := fmt.Sprintf("%s://%s:%d", defaultScheme, host, s.Port)
			clients = append(clients, client.New(url))
		}
		return clients
	}

	// Fallback: try legacy single-server discovery
	dir := os.Getenv(envDir)
	if dir == "" {
		var dirErr error
		dir, dirErr = os.Getwd()
		if dirErr != nil {
			dir = "."
		}
	}

	port, err := client.FindServerPort(dir)
	if err != nil {
		return nil
	}

	url := fmt.Sprintf("%s://%s:%d", defaultScheme, host, port)
	return []client.OpenCodeClient{client.New(url)}
}
