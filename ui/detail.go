package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/tkozakas/agent-monitor/client"
)

const (
	sessionIDPreviewLen = 12
	todoTruncateMargin  = 10
)

func renderDetail(session *client.Session, status string, todos []client.Todo, width int) string {
	if session == nil {
		return styleDim.Render("  No session selected")
	}

	var b strings.Builder

	agentLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("Agent:")
	statusLabel := lipgloss.NewStyle().Foreground(statusColor(status)).Render(
		fmt.Sprintf("%s %s", statusIcon(status), status),
	)

	title := session.Title
	if title == "" && len(session.ID) >= titleIDLen {
		title = session.ID[:titleIDLen]
	}

	b.WriteString(fmt.Sprintf("  %s %-20s Status: %s\n", agentLabel, title, statusLabel))

	if len(session.ID) >= sessionIDPreviewLen {
		b.WriteString(fmt.Sprintf("  Session: %s\n", styleDim.Render(session.ID[:sessionIDPreviewLen])))
	}

	created := time.UnixMilli(session.Time.Created)
	updated := time.UnixMilli(session.Time.Updated)
	b.WriteString(fmt.Sprintf("  Duration: %s\n", styleDim.Render(formatDuration(updated.Sub(created)))))

	if session.Summary != nil {
		s := session.Summary
		b.WriteString(fmt.Sprintf("  Changes: %s+%d %s-%d%s (%d files)\n",
			lipgloss.NewStyle().Foreground(colorGreen).Render(""),
			s.Additions,
			lipgloss.NewStyle().Foreground(colorRed).Render(""),
			s.Deletions,
			lipgloss.NewStyle().Foreground(colorFg).Render(""),
			s.Files,
		))
	}

	if len(todos) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorYellow).Render("  Todo:"))
		b.WriteString("\n")

		for _, todo := range todos {
			icon := todoIcon(todo.Status)
			clr := todoStatusColor(todo.Status)
			content := todo.Content
			maxLen := width - todoTruncateMargin
			if maxLen > 3 && len(content) > maxLen {
				content = content[:maxLen-3] + "..."
			}
			b.WriteString(fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(clr).Render(icon),
				content,
			))
		}
	}

	return b.String()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
