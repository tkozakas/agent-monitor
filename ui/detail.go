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
	metaHeaderLines     = 3
	partTextMaxLines    = 20
)

var (
	styleReasoning = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	styleToolLine  = lipgloss.NewStyle().Foreground(colorMagenta)
	styleSubtask   = lipgloss.NewStyle().Foreground(colorCyan)
	styleRole      = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
)

func renderDetail(
	session *client.Session,
	status string,
	messages []client.MessageWithParts,
	width, visibleLines, scroll int,
) string {
	if session == nil {
		return styleDim.Render("  No session selected")
	}

	var b strings.Builder

	title := session.Title
	if title == "" && len(session.ID) >= titleIDLen {
		title = session.ID[:titleIDLen]
	}

	statusLabel := lipgloss.NewStyle().Foreground(statusColor(status)).Render(
		fmt.Sprintf("%s %s", statusIcon(status), status),
	)

	created := time.UnixMilli(session.Time.Created)
	updated := time.UnixMilli(session.Time.Updated)
	duration := styleDim.Render(formatDuration(updated.Sub(created)))

	b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
		lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(title),
		statusLabel,
		duration,
	))

	if session.Summary != nil {
		s := session.Summary
		b.WriteString(fmt.Sprintf("  %s+%d %s-%d%s  %d files\n",
			lipgloss.NewStyle().Foreground(colorGreen).Render(""), s.Additions,
			lipgloss.NewStyle().Foreground(colorRed).Render(""), s.Deletions,
			lipgloss.NewStyle().Foreground(colorFg).Render(""), s.Files,
		))
	}

	b.WriteString(styleDim.Render("  "+strings.Repeat("─", clampWidth(width-4))) + "\n")

	renderMessageStream(&b, messages, width)

	allLines := strings.Split(b.String(), "\n")
	if scroll > len(allLines)-visibleLines {
		scroll = len(allLines) - visibleLines
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + visibleLines
	if end > len(allLines) {
		end = len(allLines)
	}

	return strings.Join(allLines[scroll:end], "\n")
}

func renderMessageStream(b *strings.Builder, messages []client.MessageWithParts, width int) {
	if len(messages) == 0 {
		b.WriteString(styleDim.Render("  Waiting for messages..."))
		return
	}

	for _, msg := range messages {
		role := msg.Info.Role
		agent := msg.Info.Agent
		label := role
		if agent != "" {
			label = agent
		}
		b.WriteString(fmt.Sprintf("  %s\n", styleRole.Render(label)))

		for _, p := range msg.Parts {
			renderPart(b, p, width)
		}
		b.WriteString("\n")
	}
}

func renderPart(b *strings.Builder, p client.Part, width int) {
	contentWidth := clampWidth(width - 4)

	switch p.Type {
	case partTypeText:
		lines := wrapText(p.Text, contentWidth)
		for _, line := range lines {
			b.WriteString("  " + line + "\n")
		}

	case "reasoning":
		lines := wrapText(p.Text, contentWidth)
		for _, line := range lines {
			b.WriteString("  " + styleReasoning.Render(line) + "\n")
		}

	case partTypeTool:
		name := p.Tool
		if p.State != nil && p.State.Title != "" {
			name = p.State.Title
		}
		st := ""
		if p.State != nil {
			st = p.State.Status
		}
		icon := toolStatusIcon(st)
		b.WriteString(fmt.Sprintf("  %s %s\n", styleToolLine.Render(icon+" "+name), styleDim.Render(st)))

	case partTypeSubtask:
		desc := p.Description
		if desc == "" {
			desc = p.Prompt
		}
		if len(desc) > contentWidth {
			desc = desc[:contentWidth-3] + "..."
		}
		agent := p.Agent
		if agent == "" {
			agent = "subagent"
		}
		b.WriteString(fmt.Sprintf("  %s\n", styleSubtask.Render("→ "+agent+": "+desc)))

	case "step-start":
		// ignore
	}
}

func toolStatusIcon(status string) string {
	switch status {
	case toolStatusRunning:
		return "⟳"
	case toolStatusCompleted:
		return "✓"
	case toolStatusError:
		return "✗"
	default:
		return "…"
	}
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var result []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) == 0 {
			result = append(result, "")
			continue
		}
		for len(line) > width {
			result = append(result, line[:width])
			line = line[width:]
		}
		result = append(result, line)
	}
	return result
}

func clampWidth(w int) int {
	if w < 1 {
		return 1
	}
	return w
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
