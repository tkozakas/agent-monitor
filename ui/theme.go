package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

const (
	statusBusy      = "busy"
	statusIdle      = "idle"
	statusError     = "error"
	statusRetry     = "retry"
	statusComplete  = "complete"
	statusCompleted = "completed"

	todoInProgress = "in_progress"
	todoCancelled  = "cancelled"
)

var (
	colorFg      = lipgloss.Color("#abb2bf")
	colorDim     = lipgloss.Color("#5c6370")
	colorBorder  = lipgloss.Color("#3e4452")
	colorCyan    = lipgloss.Color("#56b6c2")
	colorGreen   = lipgloss.Color("#98c379")
	colorRed     = lipgloss.Color("#e06c75")
	colorYellow  = lipgloss.Color("#e5c07b")
	colorBlue    = lipgloss.Color("#61afef")
	colorMagenta = lipgloss.Color("#c678dd")

	styleBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			PaddingLeft(1)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleDim = lipgloss.NewStyle().
			Foreground(colorDim)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorDim).
			PaddingLeft(1)
)

func statusColor(status string) color.Color {
	switch status {
	case statusBusy:
		return colorGreen
	case statusIdle:
		return colorDim
	case statusError:
		return colorRed
	case statusRetry:
		return colorYellow
	case statusComplete, statusCompleted:
		return colorBlue
	default:
		return colorFg
	}
}

func statusIcon(status string) string {
	switch status {
	case statusBusy:
		return "●"
	case statusIdle:
		return "○"
	case statusError:
		return "✗"
	case statusRetry:
		return "↻"
	case statusComplete, statusCompleted:
		return "✓"
	default:
		return "?"
	}
}

func todoIcon(status string) string {
	switch status {
	case statusCompleted:
		return "✓"
	case todoInProgress:
		return "→"
	case todoCancelled:
		return "✗"
	default:
		return "○"
	}
}

func todoStatusColor(status string) color.Color {
	switch status {
	case statusCompleted:
		return colorGreen
	case todoInProgress:
		return colorCyan
	case todoCancelled:
		return colorRed
	default:
		return colorDim
	}
}
