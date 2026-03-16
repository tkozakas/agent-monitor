package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/tkozakas/agent-monitor/client"
)

const (
	envURL  = "OPENCODE_URL"
	envPort = "OPENCODE_PORT"
	envHost = "OPENCODE_HOST"

	defaultHost   = "localhost"
	defaultScheme = "http"
	maxPanes      = 4
)

func main() {
	urls := resolveURLs()
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "no opencode servers found")
		os.Exit(1)
	}

	var sessions []rootSession
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			c := client.New(u)
			all, err := c.Sessions()
			if err != nil {
				return
			}
			mu.Lock()
			for _, s := range all {
				if s.ParentID == nil {
					sessions = append(sessions, rootSession{url: u, id: s.ID, title: s.Title, clientIdx: idx})
				}
			}
			mu.Unlock()
		}(i, url)
	}
	wg.Wait()

	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "no root sessions found")
		os.Exit(1)
	}

	if len(sessions) > maxPanes {
		sessions = sessions[:maxPanes]
	}

	selfPane := currentPaneID()
	paneIDs := make([]string, len(sessions))
	paneIDs[0] = selfPane

	for i := 1; i < len(sessions); i++ {
		target := splitTarget(i, paneIDs)
		dir := splitDir(i, len(sessions))
		args := []string{"split-window", dir, "-d", "-P", "-F", "#{pane_id}", "-t", target, "cat"}
		out, err := exec.Command("tmux", args...).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "split %d: %v\n", i, err)
			continue
		}
		paneIDs[i] = strings.TrimSpace(string(out))
	}

	if len(sessions) > 2 {
		exec.Command("tmux", "select-layout", "tiled").Run()
	}

	var sendWg sync.WaitGroup
	for i := 1; i < len(sessions); i++ {
		if paneIDs[i] == "" {
			continue
		}
		sendWg.Add(1)
		go func(pane string, s rootSession) {
			defer sendWg.Done()
			cmd := fmt.Sprintf("opencode attach %s -s %s", s.url, s.id)
			exec.Command("tmux", "respawn-pane", "-k", "-t", pane, cmd).Run()
		}(paneIDs[i], sessions[i])
	}
	sendWg.Wait()

	first := sessions[0]
	bin, err := exec.LookPath("opencode")
	if err != nil {
		fmt.Fprintln(os.Stderr, "opencode not found in PATH")
		os.Exit(1)
	}
	syscall.Exec(bin, []string{"opencode", "attach", first.url, "-s", first.id}, os.Environ())
}

func splitTarget(index int, paneIDs []string) string {
	switch index {
	case 1:
		return paneIDs[0]
	case 2:
		return paneIDs[0]
	case 3:
		return paneIDs[1]
	default:
		return paneIDs[len(paneIDs)-1]
	}
}

func splitDir(index, total int) string {
	if total <= 2 {
		return "-h"
	}
	switch index {
	case 1:
		return "-h"
	case 2, 3:
		return "-v"
	default:
		return "-h"
	}
}

func currentPaneID() string {
	out, _ := exec.Command("tmux", "display-message", "-p", "#{pane_id}").Output()
	return strings.TrimSpace(string(out))
}

type rootSession struct {
	url       string
	id        string
	title     string
	clientIdx int
}

func resolveURLs() []string {
	if url := os.Getenv(envURL); url != "" {
		return []string{url}
	}

	host := os.Getenv(envHost)
	if host == "" {
		host = defaultHost
	}

	if portStr := os.Getenv(envPort); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil && port > 0 {
			return []string{fmt.Sprintf("%s://%s:%d", defaultScheme, host, port)}
		}
	}

	servers, err := client.FindAllServerPorts()
	if err == nil && len(servers) > 0 {
		var urls []string
		for _, s := range servers {
			urls = append(urls, fmt.Sprintf("%s://%s:%d", defaultScheme, host, s.Port))
		}
		return urls
	}

	return nil
}
