package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/tkozakas/agent-monitor/client"
)

const (
	maxActivityLines   = 50
	agentNameMaxLen    = 15
	sessionIDShortLen  = 8
	summaryWidthMargin = 35
	summaryMinWidth    = 20
	descMaxLen         = 60
	textMinLen         = 5
	textMaxLen         = 80
	timeFormat         = "15:04:05"

	partTypeTool    = "tool"
	partTypeSubtask = "subtask"
	partTypeText    = "text"

	toolStatusRunning   = "running"
	toolStatusCompleted = "completed"
	toolStatusError     = "error"

	kindTool    = "tool"
	kindSubtask = "subtask"
	kindStatus  = "status"
	kindText    = "text"
)

type activity struct {
	Time      time.Time
	SessionID string
	Agent     string
	Kind      string
	Summary   string
}

func renderActivity(activities []activity, width int) string {
	if len(activities) == 0 {
		return styleDim.Render("  No activity yet")
	}

	var b strings.Builder
	start := 0
	if len(activities) > maxActivityLines {
		start = len(activities) - maxActivityLines
	}

	for _, a := range activities[start:] {
		tsStr := styleDim.Render(a.Time.Format(timeFormat))

		agent := a.Agent
		if agent == "" && len(a.SessionID) >= sessionIDShortLen {
			agent = a.SessionID[:sessionIDShortLen]
		}
		if len(agent) > agentNameMaxLen {
			agent = agent[:agentNameMaxLen]
		}

		var kindColor color.Color
		switch a.Kind {
		case kindTool:
			kindColor = colorMagenta
		case kindSubtask:
			kindColor = colorCyan
		case kindStatus:
			kindColor = colorYellow
		default:
			kindColor = colorFg
		}

		agentStr := lipgloss.NewStyle().Foreground(colorBlue).Render(agent)
		summary := a.Summary
		maxSummary := width - summaryWidthMargin
		if maxSummary < summaryMinWidth {
			maxSummary = summaryMinWidth
		}
		if len(summary) > maxSummary {
			summary = summary[:maxSummary-3] + "..."
		}
		summaryStr := lipgloss.NewStyle().Foreground(kindColor).Render(summary)

		b.WriteString(fmt.Sprintf("  %s  %-15s  %s\n", tsStr, agentStr, summaryStr))
	}

	return b.String()
}

func partToActivity(part client.Part) *activity {
	a := &activity{
		Time:      time.Now(),
		SessionID: part.SessionID,
	}

	switch part.Type {
	case partTypeTool:
		if part.State == nil {
			return nil
		}
		a.Kind = kindTool
		title := part.State.Title
		if title == "" {
			title = part.Tool
		}
		switch part.State.Status {
		case toolStatusRunning:
			a.Summary = fmt.Sprintf("tool:%s (running)", title)
		case toolStatusCompleted:
			a.Summary = fmt.Sprintf("tool:%s ✓", title)
		case toolStatusError:
			a.Summary = fmt.Sprintf("tool:%s ✗ %s", title, part.State.Error)
		default:
			return nil
		}
	case partTypeSubtask:
		a.Kind = kindSubtask
		a.Agent = part.Agent
		desc := part.Description
		if desc == "" {
			desc = part.Prompt
		}
		if len(desc) > descMaxLen {
			desc = desc[:descMaxLen-3] + "..."
		}
		a.Summary = fmt.Sprintf("→ %s: %s", part.Agent, desc)
	case partTypeText:
		if len(part.Text) < textMinLen {
			return nil
		}
		a.Kind = kindText
		text := strings.ReplaceAll(part.Text, "\n", " ")
		if len(text) > textMaxLen {
			text = text[:textMaxLen-3] + "..."
		}
		a.Summary = text
	default:
		return nil
	}

	return a
}
