package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/tkozakas/agent-monitor/client"
)

const (
	pickerWidthFraction  = 60
	pickerHeightFraction = 60
	pickerMinWidth       = 40
	pickerMinHeight      = 10
	pickerMaxVisible     = 20
	pickerTitleMaxLen    = 40
)

var (
	stylePickerBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorCyan).
				Padding(1, 2)

	stylePickerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorCyan)

	stylePickerSelected = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	stylePickerNormal = lipgloss.NewStyle().
				Foreground(colorFg)

	stylePickerDim = lipgloss.NewStyle().
			Foreground(colorDim)
)

func renderPicker(sessions []client.Session, statuses map[string]client.SessionStatus, cursor int, width, height int) string {
	boxW := width * pickerWidthFraction / 100
	if boxW < pickerMinWidth {
		boxW = pickerMinWidth
	}
	boxH := height * pickerHeightFraction / 100
	if boxH < pickerMinHeight {
		boxH = pickerMinHeight
	}

	var b strings.Builder
	b.WriteString(stylePickerTitle.Render("Select Session") + "\n")
	b.WriteString(styleDim.Render(strings.Repeat("─", boxW-6)) + "\n")

	if len(sessions) == 0 {
		b.WriteString(stylePickerDim.Render("  No sessions available"))
		return stylePickerBorder.Width(boxW - 4).Height(boxH).Render(b.String())
	}

	visible := boxH - 4
	if visible > pickerMaxVisible {
		visible = pickerMaxVisible
	}
	if visible < 1 {
		visible = 1
	}

	start := 0
	if cursor >= visible {
		start = cursor - visible + 1
	}
	end := start + visible
	if end > len(sessions) {
		end = len(sessions)
	}

	contentW := boxW - 10

	for i := start; i < end; i++ {
		s := sessions[i]
		title := s.Title
		if title == "" && len(s.ID) >= titleIDLen {
			title = s.ID[:titleIDLen]
		}
		if len(title) > pickerTitleMaxLen {
			title = title[:pickerTitleMaxLen-3] + "..."
		}

		st := statusIdle
		if status, ok := statuses[s.ID]; ok {
			st = status.Type
		}

		created := time.UnixMilli(s.Time.Created)
		updated := time.UnixMilli(s.Time.Updated)
		dur := formatDuration(updated.Sub(created))

		icon := statusIcon(st)
		clr := statusColor(st)
		statusStr := lipgloss.NewStyle().Foreground(clr).Render(icon + " " + st)

		line := fmt.Sprintf("%s  %s  %s", title, statusStr, stylePickerDim.Render(dur))
		if len(line) > contentW {
			line = line[:contentW]
		}

		if i == cursor {
			b.WriteString(stylePickerSelected.Render("❯ " + line))
		} else {
			b.WriteString(stylePickerNormal.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString(stylePickerDim.Render(fmt.Sprintf("\n  %d/%d sessions  j/k nav  enter select  esc close", cursor+1, len(sessions))))

	return stylePickerBorder.Width(boxW - 4).Height(boxH).Render(b.String())
}
