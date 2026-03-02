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
	serverURL := resolveServerURL()
	oc := client.New(serverURL)

	p := tea.NewProgram(ui.New(oc))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func resolveServerURL() string {
	if url := os.Getenv(envURL); url != "" {
		return url
	}

	host := os.Getenv(envHost)
	if host == "" {
		host = defaultHost
	}

	if portStr := os.Getenv(envPort); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil && port > 0 {
			return fmt.Sprintf("%s://%s:%d", defaultScheme, host, port)
		}
	}

	dir := os.Getenv(envDir)
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			dir = "."
		}
	}

	port, err := client.FindServerPort(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not find opencode server: %v\nset %s or start opencode first\n", err, envURL)
		os.Exit(1)
	}

	return fmt.Sprintf("%s://%s:%d", defaultScheme, host, port)
}
