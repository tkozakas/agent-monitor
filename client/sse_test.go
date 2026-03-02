package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSEParseWellFormedStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		events := []string{
			`{"type":"session.status","properties":{"sessionID":"s1","status":{"type":"busy"}}}`,
			`{"type":"session.created","properties":{"info":{"id":"s2","title":"new"}}}`,
		}
		for _, e := range events {
			fmt.Fprintf(w, "data: %s\n\n", e)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.sse = newSSEClient(srv.URL)
	c.StartEvents()
	defer c.StopEvents()

	timeout := time.After(2 * time.Second)
	var received []Event
	for i := 0; i < 2; i++ {
		select {
		case ev := <-c.Events():
			received = append(received, ev)
		case <-timeout:
			t.Fatalf("timed out waiting for event %d", i+1)
		}
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}
	if received[0].Type != EventSessionStatus {
		t.Errorf("expected first event type %s, got %s", EventSessionStatus, received[0].Type)
	}
	if received[1].Type != EventSessionCreated {
		t.Errorf("expected second event type %s, got %s", EventSessionCreated, received[1].Type)
	}
}

func TestSSEStopTerminatesGoroutine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		// Send events slowly until client disconnects
		for {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
				fmt.Fprintf(w, "data: {\"type\":\"ping\",\"properties\":{}}\n\n")
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.sse = newSSEClient(srv.URL)
	c.StartEvents()

	// Read at least one event to confirm connection
	select {
	case <-c.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	// Stop should return quickly
	done := make(chan struct{})
	go func() {
		c.StopEvents()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("StopEvents did not return in time")
	}
}

func TestSSEDoubleStopNoPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.sse = newSSEClient(srv.URL)
	c.StartEvents()

	// Double stop should not panic
	c.StopEvents()
	c.StopEvents()
}

func TestSSEMultiLineData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		// Multi-line data event
		fmt.Fprint(w, "data: {\"type\":\"test\",\n")
		fmt.Fprint(w, "data: \"properties\":{}}\n")
		fmt.Fprint(w, "\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.sse = newSSEClient(srv.URL)
	c.StartEvents()
	defer c.StopEvents()

	select {
	case ev := <-c.Events():
		if ev.Type != "test" {
			t.Errorf("expected type 'test', got %s", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for multi-line event")
	}
}
