package client

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	sseEventBufferSize   = 256
	sseReconnectDelay    = 2 * time.Second
	sseScannerBufferSize = 512 * 1024
	sseDataPrefix        = "data: "
)

// SSE event type constants used by the opencode server.
const (
	EventSessionStatus     = "session.status"
	EventSessionCreated    = "session.created"
	EventSessionUpdated    = "session.updated"
	EventMessagePartUpdate = "message.part.updated"
	EventTodoUpdated       = "todo.updated"
)

// Event represents a single server-sent event from the opencode server.
type Event struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

// SessionStatusEvent carries a status change for a session.
type SessionStatusEvent struct {
	SessionID string        `json:"sessionID"`
	Status    SessionStatus `json:"status"`
}

// MessagePartEvent carries a message part update with an optional delta.
type MessagePartEvent struct {
	Part  Part   `json:"part"`
	Delta string `json:"delta,omitempty"`
}

// SessionEvent carries session creation or update information.
type SessionEvent struct {
	Info Session `json:"info"`
}

// TodoEvent carries updated todo items for a session.
type TodoEvent struct {
	SessionID string `json:"sessionID"`
	Todos     []Todo `json:"todos"`
}

type sseClient struct {
	url    string
	events chan Event
	client *http.Client
	cancel context.CancelFunc
	once   sync.Once
}

func newSSEClient(url string) *sseClient {
	return &sseClient{
		url:    url,
		events: make(chan Event, sseEventBufferSize),
		client: &http.Client{Timeout: 0},
	}
}

func (s *sseClient) start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.connect(ctx)
}

func (s *sseClient) stop() {
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
}

func (s *sseClient) connect(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.stream(ctx); err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sseReconnectDelay):
			}
		}
	}
}

func (s *sseClient) stream(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, sseScannerBufferSize), sseScannerBufferSize)

	var dataLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Text()

		if strings.HasPrefix(line, sseDataPrefix) {
			dataLines = append(dataLines, strings.TrimPrefix(line, sseDataPrefix))
			continue
		}

		if line == "" && len(dataLines) > 0 {
			data := strings.Join(dataLines, "\n")
			dataLines = dataLines[:0]

			var event Event
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			select {
			case s.events <- event:
			default:
			}
		}
	}

	return scanner.Err()
}
