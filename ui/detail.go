package ui

import (
	"encoding/json"
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
	sparklineWidth      = 20
	toolExpandMaxLines  = 10
)

var (
	styleReasoning = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	styleToolLine  = lipgloss.NewStyle().Foreground(colorMagenta)
	styleSubtask   = lipgloss.NewStyle().Foreground(colorCyan)
	styleRole      = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	styleCostLine  = lipgloss.NewStyle().Foreground(colorDim)

	styleSparkIn    = lipgloss.NewStyle().Foreground(colorBlue)
	styleSparkOut   = lipgloss.NewStyle().Foreground(colorGreen)
	styleSparkCache = lipgloss.NewStyle().Foreground(colorDim)
)

func renderDetail(
	session *client.Session,
	status string,
	messages []client.MessageWithParts,
	width, visibleLines, scroll int,
	expandTools bool,
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

	totalCost, totalIn, totalOut, totalReasoning, totalCache := aggregateCost(messages)
	if totalCost > 0 || totalIn > 0 || totalOut > 0 {
		costStr := fmt.Sprintf("  $%.2f  %s in  %s out",
			totalCost, formatTokens(totalIn), formatTokens(totalOut))
		if totalReasoning > 0 {
			costStr += fmt.Sprintf("  %s reason", formatTokens(totalReasoning))
		}
		if totalCache > 0 {
			costStr += fmt.Sprintf("  %s cache", formatTokens(totalCache))
		}
		b.WriteString(styleCostLine.Render(costStr) + "\n")
		b.WriteString("  " + renderSparkline(totalIn, totalOut, totalCache, sparklineWidth) + "\n")
	}

	if session.Summary != nil {
		s := session.Summary
		b.WriteString(fmt.Sprintf("  %s+%d %s-%d%s  %d files\n",
			lipgloss.NewStyle().Foreground(colorGreen).Render(""), s.Additions,
			lipgloss.NewStyle().Foreground(colorRed).Render(""), s.Deletions,
			lipgloss.NewStyle().Foreground(colorFg).Render(""), s.Files,
		))
	}

	b.WriteString(styleDim.Render("  "+strings.Repeat("─", clampWidth(width-4))) + "\n")

	renderMessageStream(&b, messages, width, expandTools)

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

func aggregateCost(messages []client.MessageWithParts) (totalCost float64, totalIn, totalOut, totalReasoning, totalCacheRead int) {
	for _, msg := range messages {
		totalCost += msg.Info.Cost
		if msg.Info.Tokens != nil {
			totalIn += msg.Info.Tokens.Input
			totalOut += msg.Info.Tokens.Output
			totalReasoning += msg.Info.Tokens.Reasoning
			totalCacheRead += msg.Info.Tokens.Cache.Read
		}
	}
	return
}

func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

func renderSparkline(in, out, cache, width int) string {
	total := in + out + cache
	if total == 0 || width <= 0 {
		return ""
	}

	inW := in * width / total
	outW := out * width / total
	cacheW := width - inW - outW
	if cacheW < 0 {
		cacheW = 0
	}

	var b strings.Builder
	b.WriteString("[")
	b.WriteString(styleSparkIn.Render(strings.Repeat("█", inW)))
	b.WriteString(styleSparkOut.Render(strings.Repeat("█", outW)))
	b.WriteString(styleSparkCache.Render(strings.Repeat("░", cacheW)))
	b.WriteString("]")

	inPct := in * 100 / total
	outPct := out * 100 / total
	cachePct := cache * 100 / total
	b.WriteString(fmt.Sprintf(" in:%d%% out:%d%% cache:%d%%", inPct, outPct, cachePct))

	return b.String()
}

func renderMessageStream(b *strings.Builder, messages []client.MessageWithParts, width int, expandTools bool) {
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
			renderPart(b, p, width, expandTools)
		}
		b.WriteString("\n")
	}
}

func renderPart(b *strings.Builder, p client.Part, width int, expandTools bool) {
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

		if expandTools && p.State != nil {
			renderToolExpanded(b, p.State, contentWidth)
		}

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

func renderToolExpanded(b *strings.Builder, state *client.ToolState, width int) {
	if len(state.Input) > 0 {
		formatted := formatJSON(state.Input, width)
		lines := strings.Split(formatted, "\n")
		if len(lines) > toolExpandMaxLines {
			lines = lines[:toolExpandMaxLines]
			lines = append(lines, "    ...")
		}
		for _, line := range lines {
			b.WriteString("    " + styleDim.Render(line) + "\n")
		}
	}
	if state.Output != "" {
		lines := strings.Split(state.Output, "\n")
		if len(lines) > toolExpandMaxLines {
			lines = lines[:toolExpandMaxLines]
			lines = append(lines, "    ...")
		}
		for _, line := range lines {
			truncated := line
			if len(truncated) > width {
				truncated = truncated[:width-3] + "..."
			}
			b.WriteString("    " + truncated + "\n")
		}
	}
}

func formatJSON(raw json.RawMessage, width int) string {
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		s := string(raw)
		if len(s) > width {
			s = s[:width-3] + "..."
		}
		return s
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(formatted)
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
