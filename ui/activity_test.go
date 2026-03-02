package ui

import (
	"testing"

	"github.com/tkozakas/agent-monitor/client"
)

func TestPartToActivityTool(t *testing.T) {
	tests := []struct {
		name    string
		part    client.Part
		wantNil bool
		kind    string
	}{
		{
			name: "completedTool",
			part: client.Part{
				Type: "tool", SessionID: "s1", Tool: "edit",
				State: &client.ToolState{Status: "completed", Title: "Edit file"},
			},
			kind: "tool",
		},
		{
			name: "runningTool",
			part: client.Part{
				Type: "tool", SessionID: "s1", Tool: "bash",
				State: &client.ToolState{Status: "running", Title: "Run tests"},
			},
			kind: "tool",
		},
		{
			name: "errorTool",
			part: client.Part{
				Type: "tool", SessionID: "s1", Tool: "bash",
				State: &client.ToolState{Status: "error", Title: "Fail", Error: "exit 1"},
			},
			kind: "tool",
		},
		{
			name:    "pendingToolIgnored",
			part:    client.Part{Type: "tool", State: &client.ToolState{Status: "pending"}},
			wantNil: true,
		},
		{
			name:    "toolWithNilState",
			part:    client.Part{Type: "tool"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := partToActivity(tt.part)
			if tt.wantNil {
				if a != nil {
					t.Errorf("expected nil, got %+v", a)
				}
				return
			}
			if a == nil {
				t.Fatal("expected activity, got nil")
			}
			if a.Kind != tt.kind {
				t.Errorf("expected kind %s, got %s", tt.kind, a.Kind)
			}
		})
	}
}

func TestPartToActivitySubtask(t *testing.T) {
	a := partToActivity(client.Part{
		Type: "subtask", SessionID: "s1",
		Agent: "coder", Description: "implement feature",
	})
	if a == nil {
		t.Fatal("expected activity")
	}
	if a.Kind != "subtask" {
		t.Errorf("expected kind subtask, got %s", a.Kind)
	}
	if a.Agent != "coder" {
		t.Errorf("expected agent coder, got %s", a.Agent)
	}
}

func TestPartToActivityTextShort(t *testing.T) {
	a := partToActivity(client.Part{Type: "text", Text: "hi"})
	if a != nil {
		t.Error("expected nil for short text")
	}
}

func TestPartToActivityTextLong(t *testing.T) {
	a := partToActivity(client.Part{Type: "text", SessionID: "s1", Text: "this is a longer message"})
	if a == nil {
		t.Fatal("expected activity")
	}
	if a.Kind != "text" {
		t.Errorf("expected kind text, got %s", a.Kind)
	}
}

func TestPartToActivityUnknownType(t *testing.T) {
	a := partToActivity(client.Part{Type: "step-start"})
	if a != nil {
		t.Error("expected nil for unknown type")
	}
}
