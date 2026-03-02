package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tkozakas/agent-monitor/client"
)

const (
	envRefreshInterval = "AGENT_MONITOR_REFRESH"

	defaultRefreshInterval = 5 * time.Second

	treeWidthPercent  = 35
	topHeightPercent  = 55
	layoutPadding     = 4
	layoutBottomExtra = 5
	panelBorderOffset = 2
	panelCount        = 3

	maxActivities = 200

	helpText = "q quit  j/k nav  tab panel  a abort  r refresh"
)

type panel int

const (
	panelTree panel = iota
	panelDetail
	panelActivity
)

// Model is the top-level Bubble Tea model for the agent monitor TUI.
type Model struct {
	oc              client.OpenCodeClient
	width           int
	height          int
	focus           panel
	sessions        []client.Session
	statuses        map[string]client.SessionStatus
	todos           map[string][]client.Todo
	tree            []*treeNode
	flatIDs         []string
	cursor          int
	activities      []activity
	err             error
	refreshInterval time.Duration
}

type (
	refreshMsg struct {
		sessions []client.Session
		statuses map[string]client.SessionStatus
	}
	sseMsg   struct{ event client.Event }
	todosMsg struct {
		sessionID string
		todos     []client.Todo
	}
	errMsg  struct{ err error }
	tickMsg struct{}
)

// New creates a new Model using the given OpenCodeClient.
func New(oc client.OpenCodeClient) Model {
	return Model{
		oc:              oc,
		statuses:        make(map[string]client.SessionStatus),
		todos:           make(map[string][]client.Todo),
		refreshInterval: parseRefreshInterval(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchSessions(m.oc),
		startSSE(m.oc),
		tick(m.refreshInterval),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	case refreshMsg:
		m.sessions = msg.sessions
		m.statuses = msg.statuses
		m = m.rebuildTree()
		return m, fetchTodos(m.oc, m.selectedID())
	case sseMsg:
		return m.handleSSE(msg.event)
	case todosMsg:
		m.todos[msg.sessionID] = msg.todos
	case tickMsg:
		return m, tea.Batch(fetchSessions(m.oc), tick(m.refreshInterval))
	case errMsg:
		m.err = msg.err
	}
	return m, nil
}

func (m Model) View() tea.View {
	v := tea.View{AltScreen: true}

	if m.width == 0 || m.height == 0 {
		v.Content = "Loading..."
		return v
	}

	treeW := m.width * treeWidthPercent / 100
	detailW := m.width - treeW - layoutPadding
	topH := m.height*topHeightPercent/100 - panelBorderOffset
	bottomH := m.height - topH - layoutBottomExtra

	treePanel := panelBox("Agents",
		renderTree(m.tree, m.selectedID(), treeW-layoutPadding),
		treeW, topH, m.focus == panelTree,
	)

	var (
		sess   *client.Session
		status string
	)
	if id := m.selectedID(); id != "" {
		for i := range m.sessions {
			if m.sessions[i].ID == id {
				sess = &m.sessions[i]
				if st, ok := m.statuses[id]; ok {
					status = st.Type
				}
				break
			}
		}
	}

	detailPanel := panelBox("Details",
		renderDetail(sess, status, m.todos[m.selectedID()], detailW-layoutPadding),
		detailW, topH, m.focus == panelDetail,
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, treePanel, detailPanel)
	actPanel := panelBox("Activity",
		renderActivity(m.activities, m.width-layoutPadding),
		m.width-panelBorderOffset, bottomH, m.focus == panelActivity,
	)

	help := styleHelp.Render(helpText)
	if m.err != nil {
		help = lipgloss.NewStyle().Foreground(colorRed).PaddingLeft(1).
			Render(fmt.Sprintf("Error: %v", m.err))
	}

	v.Content = lipgloss.JoinVertical(lipgloss.Left, topRow, actPanel, help)
	return v
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.oc.StopEvents()
		return m, tea.Quit
	case "j", "down":
		m.cursor++
		m = m.clampedCursor()
		return m, fetchTodos(m.oc, m.selectedID())
	case "k", "up":
		m.cursor--
		m = m.clampedCursor()
		return m, fetchTodos(m.oc, m.selectedID())
	case "tab":
		m.focus = (m.focus + 1) % panelCount
	case "shift+tab":
		m.focus = (m.focus + panelCount - 1) % panelCount
	case "a":
		if id := m.selectedID(); id != "" {
			return m, abortSession(m.oc, id)
		}
	case "r":
		return m, fetchSessions(m.oc)
	}
	return m, nil
}

func (m Model) handleSSE(event client.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case client.EventSessionStatus:
		var e client.SessionStatusEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			m.statuses[e.SessionID] = e.Status
			m = m.rebuildTree()
			m.activities = append(m.activities, activity{
				Time: time.Now(), SessionID: e.SessionID,
				Kind: kindStatus, Summary: fmt.Sprintf("status → %s", e.Status.Type),
			})
			m = m.trimActivities()
		}
	case client.EventSessionCreated:
		var e client.SessionEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			m.sessions = append(m.sessions, e.Info)
			m = m.rebuildTree()
			m.activities = append(m.activities, activity{
				Time: time.Now(), SessionID: e.Info.ID,
				Kind: kindStatus, Summary: fmt.Sprintf("session created: %s", e.Info.Title),
			})
			m = m.trimActivities()
		}
	case client.EventSessionUpdated:
		var e client.SessionEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			for i := range m.sessions {
				if m.sessions[i].ID == e.Info.ID {
					m.sessions[i] = e.Info
					break
				}
			}
			m = m.rebuildTree()
		}
	case client.EventMessagePartUpdate:
		var e client.MessagePartEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			if a := partToActivity(e.Part); a != nil {
				m.activities = append(m.activities, *a)
				m = m.trimActivities()
			}
		}
	case client.EventTodoUpdated:
		var e client.TodoEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			m.todos[e.SessionID] = e.Todos
		}
	}
	return m, listenSSE(m.oc)
}

func (m Model) rebuildTree() Model {
	m.tree = buildTree(m.sessions, m.statuses)
	m.flatIDs = flattenTree(m.tree)
	m = m.clampedCursor()
	return m
}

func (m Model) selectedID() string {
	if len(m.flatIDs) == 0 {
		return ""
	}
	return m.flatIDs[m.cursor]
}

func (m Model) clampedCursor() Model {
	max := len(m.flatIDs) - 1
	if max < 0 {
		max = 0
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > max {
		m.cursor = max
	}
	return m
}

func (m Model) trimActivities() Model {
	if len(m.activities) > maxActivities {
		m.activities = m.activities[len(m.activities)-maxActivities:]
	}
	return m
}

func panelBox(title, content string, width, height int, focused bool) string {
	border := styleBorder
	if focused {
		border = border.BorderForeground(colorCyan)
	}

	lines := strings.Split(content, "\n")
	if len(lines) > height-panelBorderOffset {
		lines = lines[:height-panelBorderOffset]
	}

	return border.Width(width - panelBorderOffset).Height(height).
		Render(styleTitle.Render(title) + "\n" + strings.Join(lines, "\n"))
}

func fetchSessions(oc client.OpenCodeClient) tea.Cmd {
	return func() tea.Msg {
		sessions, err := oc.Sessions()
		if err != nil {
			return errMsg{err}
		}
		statuses, err := oc.SessionStatuses()
		if err != nil {
			return errMsg{err}
		}
		return refreshMsg{sessions: sessions, statuses: statuses}
	}
}

func startSSE(oc client.OpenCodeClient) tea.Cmd {
	return func() tea.Msg {
		oc.StartEvents()
		event, ok := <-oc.Events()
		if !ok {
			return nil
		}
		return sseMsg{event: event}
	}
}

func listenSSE(oc client.OpenCodeClient) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-oc.Events()
		if !ok {
			return nil
		}
		return sseMsg{event: event}
	}
}

func fetchTodos(oc client.OpenCodeClient, sessionID string) tea.Cmd {
	if sessionID == "" {
		return nil
	}
	return func() tea.Msg {
		todos, err := oc.SessionTodos(sessionID)
		if err != nil {
			return nil
		}
		return todosMsg{sessionID: sessionID, todos: todos}
	}
}

func abortSession(oc client.OpenCodeClient, id string) tea.Cmd {
	return func() tea.Msg {
		if err := oc.Abort(id); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func tick(interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(interval)
		return tickMsg{}
	}
}

func parseRefreshInterval() time.Duration {
	if v := os.Getenv(envRefreshInterval); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultRefreshInterval
}
