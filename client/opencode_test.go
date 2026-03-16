package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	c := New("http://localhost:4096")
	if c.base != "http://localhost:4096" {
		t.Errorf("expected base http://localhost:4096, got %s", c.base)
	}
	if c.sse == nil {
		t.Error("expected SSE client to be initialized")
	}
}

func TestClientImplementsInterface(t *testing.T) {
	var _ OpenCodeClient = (*Client)(nil)
}

func TestSessions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]Session{
			{ID: "s1", Title: "test session"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	sessions, err := c.Sessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != "s1" {
		t.Errorf("expected id s1, got %s", sessions[0].ID)
	}
}

func TestSessionStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]SessionStatus{
			"s1": {Type: "busy"},
			"s2": {Type: "idle"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	statuses, err := c.SessionStatuses()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statuses["s1"].Type != "busy" {
		t.Errorf("expected s1 busy, got %s", statuses["s1"].Type)
	}
	if statuses["s2"].Type != "idle" {
		t.Errorf("expected s2 idle, got %s", statuses["s2"].Type)
	}
}

func TestSessionChildren(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session/parent1/children" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]Session{
			{ID: "child1", Title: "child session"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	children, err := c.SessionChildren("parent1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(children) != 1 || children[0].ID != "child1" {
		t.Errorf("expected 1 child with id child1, got %v", children)
	}
}

func TestSessionTodos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Todo{
			{ID: "t1", Content: "fix bug", Status: "in_progress", Priority: "high"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	todos, err := c.SessionTodos("s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 1 || todos[0].Content != "fix bug" {
		t.Errorf("expected 1 todo 'fix bug', got %v", todos)
	}
}

func TestAbort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/session/s1/abort" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.Abort("s1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAbortError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.Abort("s1"); err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Sessions()
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestSendMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/session/s1/prompt_async" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.SendMessage("s1", "hello world"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendMessageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.SendMessage("s1", "hello"); err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestCreateSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(Session{ID: "new-session", Title: "New"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	sess, err := c.CreateSession()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID != "new-session" {
		t.Errorf("expected id new-session, got %s", sess.ID)
	}
}

func TestBaseURL(t *testing.T) {
	c := New("http://localhost:9999")
	if c.BaseURL() != "http://localhost:9999" {
		t.Errorf("expected http://localhost:9999, got %s", c.BaseURL())
	}
}
