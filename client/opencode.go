package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	httpTimeout = 10 * time.Second

	pathSessions      = "/session"
	pathSessionStatus = "/session/status"
	pathSessionByID   = "/session/%s"
	pathChildren      = "/session/%s/children"
	pathMessages      = "/session/%s/message"
	pathTodos         = "/session/%s/todo"
	pathAbort         = "/session/%s/abort"
	pathAgents        = "/app/agents"
	pathEvents        = "/event"
)

// OpenCodeClient defines the interface for interacting with an opencode server.
type OpenCodeClient interface {
	Sessions() ([]Session, error)
	SessionStatuses() (map[string]SessionStatus, error)
	SessionChildren(id string) ([]Session, error)
	SessionMessages(id string) ([]MessageWithParts, error)
	SessionTodos(id string) ([]Todo, error)
	Agents() ([]AgentInfo, error)
	Abort(sessionID string) error
	ReadDelegations() ([]Delegation, error)
	Events() <-chan Event
	StartEvents()
	StopEvents()
}

// Client is the concrete implementation of OpenCodeClient that talks to an
// opencode HTTP server and streams events via SSE.
type Client struct {
	base string
	http *http.Client
	sse  *sseClient
}

// Session represents an opencode coding session.
type Session struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"projectID"`
	Directory string  `json:"directory"`
	ParentID  *string `json:"parentID,omitempty"`
	Title     string  `json:"title"`
	Version   string  `json:"version"`
	Time      struct {
		Created    int64  `json:"created"`
		Updated    int64  `json:"updated"`
		Compacting *int64 `json:"compacting,omitempty"`
	} `json:"time"`
	Summary *SessionSummary `json:"summary,omitempty"`
}

// SessionSummary holds aggregated change stats for a session.
type SessionSummary struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

// SessionStatus carries the current status type and optional metadata for a session.
type SessionStatus struct {
	Type    string `json:"type"`
	Attempt int    `json:"attempt,omitempty"`
	Message string `json:"message,omitempty"`
}

// MessageWithParts wraps a message with its content parts as returned by the API.
type MessageWithParts struct {
	Info  Message `json:"info"`
	Parts []Part  `json:"parts"`
}

// Message represents a single message within a session conversation.
type Message struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Time      struct {
		Created   int64  `json:"created"`
		Completed *int64 `json:"completed,omitempty"`
	} `json:"time"`
	ParentID   string  `json:"parentID,omitempty"`
	ModelID    string  `json:"modelID,omitempty"`
	ProviderID string  `json:"providerID,omitempty"`
	Mode       string  `json:"mode,omitempty"`
	Cost       float64 `json:"cost,omitempty"`
	Tokens     *Tokens `json:"tokens,omitempty"`
	Agent      string  `json:"agent,omitempty"`
}

// Tokens carries token usage statistics for a message.
type Tokens struct {
	Input     int `json:"input"`
	Output    int `json:"output"`
	Reasoning int `json:"reasoning"`
	Cache     struct {
		Read  int `json:"read"`
		Write int `json:"write"`
	} `json:"cache"`
}

// Todo represents a task item tracked within a session.
type Todo struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// Part represents a message part such as text, tool call, or subtask.
type Part struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"sessionID"`
	MessageID   string     `json:"messageID"`
	Type        string     `json:"type"`
	Text        string     `json:"text,omitempty"`
	CallID      string     `json:"callID,omitempty"`
	Tool        string     `json:"tool,omitempty"`
	State       *ToolState `json:"state,omitempty"`
	Prompt      string     `json:"prompt,omitempty"`
	Description string     `json:"description,omitempty"`
	Agent       string     `json:"agent,omitempty"`
}

// ToolState holds the execution state and result of a tool invocation.
type ToolState struct {
	Status string          `json:"status"`
	Title  string          `json:"title,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
	Output string          `json:"output,omitempty"`
	Error  string          `json:"error,omitempty"`
	Time   *ToolTime       `json:"time,omitempty"`
}

// ToolTime records the start and optional end timestamps of a tool execution.
type ToolTime struct {
	Start int64  `json:"start"`
	End   *int64 `json:"end,omitempty"`
}

// AgentInfo describes a registered agent in the opencode system.
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode"`
	BuiltIn     bool   `json:"builtIn"`
	Color       string `json:"color,omitempty"`
}

// New creates a new Client connected to the given base URL.
func New(baseURL string) *Client {
	c := &Client{
		base: baseURL,
		http: &http.Client{Timeout: httpTimeout},
	}
	c.sse = newSSEClient(baseURL + pathEvents)
	return c
}

// Sessions returns all sessions from the opencode server.
func (c *Client) Sessions() ([]Session, error) {
	return get[[]Session](c, pathSessions)
}

// SessionStatuses returns a map of session ID to current status.
func (c *Client) SessionStatuses() (map[string]SessionStatus, error) {
	return get[map[string]SessionStatus](c, pathSessionStatus)
}

// SessionChildren returns the child sessions of the given parent session.
func (c *Client) SessionChildren(id string) ([]Session, error) {
	return get[[]Session](c, fmt.Sprintf(pathChildren, id))
}

// SessionMessages returns all messages with their parts for a session.
func (c *Client) SessionMessages(id string) ([]MessageWithParts, error) {
	return get[[]MessageWithParts](c, fmt.Sprintf(pathMessages, id))
}

// SessionTodos returns all todo items for a session.
func (c *Client) SessionTodos(id string) ([]Todo, error) {
	return get[[]Todo](c, fmt.Sprintf(pathTodos, id))
}

// Agents returns all registered agents.
func (c *Client) Agents() ([]AgentInfo, error) {
	return get[[]AgentInfo](c, pathAgents)
}

// Abort requests the server to abort the given session.
func (c *Client) Abort(sessionID string) error {
	url := c.base + fmt.Sprintf(pathAbort, sessionID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("abort failed: %d", resp.StatusCode)
	}
	return nil
}

// Events returns the channel of server-sent events.
func (c *Client) Events() <-chan Event {
	return c.sse.events
}

// StartEvents begins streaming server-sent events from the opencode server.
func (c *Client) StartEvents() {
	c.sse.start()
}

// StopEvents stops the SSE listener. It is safe to call multiple times.
func (c *Client) StopEvents() {
	c.sse.stop()
}

func get[T any](c *Client, path string) (T, error) {
	var zero T
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, err
	}
	return result, nil
}
