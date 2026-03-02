package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	messages        map[string][]client.MessageWithParts
	tree            []*treeNode
	flatIDs         []string
	cursor          int
	detailScroll    int
	rootSessionID   string
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
	messagesMsg struct {
		sessionID string
		messages  []client.MessageWithParts
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
		messages:        make(map[string][]client.MessageWithParts),
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
		m = m.resolveRoot()
		m = m.filterToTree()
		m = m.rebuildTree()
		return m, tea.Batch(
			fetchTodos(m.oc, m.selectedID()),
			fetchMessages(m.oc, m.selectedID()),
		)
	case sseMsg:
		return m.handleSSE(msg.event)
	case todosMsg:
		m.todos[msg.sessionID] = msg.todos
	case messagesMsg:
		m.messages[msg.sessionID] = msg.messages
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

	msgs := m.messages[m.selectedID()]
	detailPanel := panelBox("Details",
		renderDetail(sess, status, msgs, detailW-layoutPadding, topH-panelBorderOffset-1, m.detailScroll),
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
		if m.focus == panelDetail {
			m.detailScroll++
			return m, nil
		}
		m.cursor++
		m = m.clampedCursor()
		m.detailScroll = 0
		return m, tea.Batch(
			fetchTodos(m.oc, m.selectedID()),
			fetchMessages(m.oc, m.selectedID()),
		)
	case "k", "up":
		if m.focus == panelDetail {
			m.detailScroll--
			if m.detailScroll < 0 {
				m.detailScroll = 0
			}
			return m, nil
		}
		m.cursor--
		m = m.clampedCursor()
		m.detailScroll = 0
		return m, tea.Batch(
			fetchTodos(m.oc, m.selectedID()),
			fetchMessages(m.oc, m.selectedID()),
		)
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
			if m.isInTree(e.SessionID) {
				m.statuses[e.SessionID] = e.Status
				m = m.rebuildTree()
				m.activities = append(m.activities, activity{
					Time: time.Now(), SessionID: e.SessionID,
					Kind: kindStatus, Summary: fmt.Sprintf("status → %s", e.Status.Type),
				})
				m = m.trimActivities()
			}
		}
	case client.EventSessionCreated:
		var e client.SessionEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			if m.isDescendant(e.Info) {
				m.sessions = append(m.sessions, e.Info)
				m = m.rebuildTree()
				m.activities = append(m.activities, activity{
					Time: time.Now(), SessionID: e.Info.ID,
					Kind: kindStatus, Summary: fmt.Sprintf("session created: %s", e.Info.Title),
				})
				m = m.trimActivities()
			}
		}
	case client.EventSessionUpdated:
		var e client.SessionEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			if m.isInTree(e.Info.ID) {
				for i := range m.sessions {
					if m.sessions[i].ID == e.Info.ID {
						m.sessions[i] = e.Info
						break
					}
				}
				m = m.rebuildTree()
			}
		}
	case client.EventMessagePartUpdate:
		var e client.MessagePartEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			if m.isInTree(e.Part.SessionID) {
				if a := partToActivity(e.Part); a != nil {
					m.activities = append(m.activities, *a)
					m = m.trimActivities()
				}
				if e.Part.SessionID == m.selectedID() {
					return m, tea.Batch(
						listenSSE(m.oc),
						fetchMessages(m.oc, e.Part.SessionID),
					)
				}
			}
		}
	case client.EventTodoUpdated:
		var e client.TodoEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			if m.isInTree(e.SessionID) {
				m.todos[e.SessionID] = e.Todos
			}
		}
	}
	return m, listenSSE(m.oc)
}

func (m Model) resolveRoot() Model {
	if len(m.sessions) == 0 {
		return m
	}

	sorted := make([]client.Session, len(m.sessions))
	copy(sorted, m.sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Time.Created > sorted[j].Time.Created
	})

	// Find the most recent busy session.
	for _, s := range sorted {
		if s.ParentID != nil {
			continue
		}
		if st, ok := m.statuses[s.ID]; ok && st.Type == statusBusy {
			m.rootSessionID = s.ID
			return m
		}
	}

	// Fall back to the most recent root session.
	for _, s := range sorted {
		if s.ParentID == nil {
			m.rootSessionID = s.ID
			return m
		}
	}

	m.rootSessionID = sorted[0].ID
	return m
}

func (m Model) filterToTree() Model {
	if m.rootSessionID == "" {
		return m
	}

	idSet := m.treeIDs()
	var filtered []client.Session
	for _, s := range m.sessions {
		if idSet[s.ID] {
			filtered = append(filtered, s)
		}
	}
	m.sessions = filtered
	return m
}

func (m Model) treeIDs() map[string]bool {
	ids := map[string]bool{m.rootSessionID: true}
	changed := true
	for changed {
		changed = false
		for _, s := range m.sessions {
			if s.ParentID != nil && ids[*s.ParentID] && !ids[s.ID] {
				ids[s.ID] = true
				changed = true
			}
		}
	}
	return ids
}

func (m Model) isInTree(sessionID string) bool {
	if m.rootSessionID == "" {
		return true
	}
	return m.treeIDs()[sessionID]
}

func (m Model) isDescendant(s client.Session) bool {
	if m.rootSessionID == "" {
		return true
	}
	if s.ParentID == nil {
		return s.ID == m.rootSessionID
	}
	return m.treeIDs()[*s.ParentID]
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

func fetchMessages(oc client.OpenCodeClient, sessionID string) tea.Cmd {
	if sessionID == "" {
		return nil
	}
	return func() tea.Msg {
		msgs, err := oc.SessionMessages(sessionID)
		if err != nil {
			return nil
		}
		return messagesMsg{sessionID: sessionID, messages: msgs}
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
