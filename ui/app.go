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

	sidebarWidthPercent = 22
	layoutPadding       = 4
	panelBorderOffset   = 2

	helpBarHeight = 3
	maxScrollVal  = 999999

	focusSidebar = -1

	helpText = "q quit  j/k nav  Tab focus  Enter open  x close  i prompt  m swap  G follow  f filter  a abort  r refresh"
)

// Model is the top-level Bubble Tea model for the agent monitor TUI.
type Model struct {
	clients         []client.OpenCodeClient
	width           int
	height          int
	allSessions     []sessionEntry
	statuses        map[string]client.SessionStatus
	messages        map[string][]client.MessageWithParts
	agentNames      map[string]string
	tree            []*treeNode
	flatIDs         []string
	sidebarCursor   int
	showAll         bool
	panes           []Pane
	focusedPane     int // -1 = sidebar
	swapMark        int // -1 = none
	err             error
	refreshInterval time.Duration
}

type (
	refreshMsg struct {
		clientIdx int
		sessions  []client.Session
		statuses  map[string]client.SessionStatus
	}
	sseMsg struct {
		clientIdx int
		event     client.Event
	}
	messagesMsg struct {
		sessionID string
		messages  []client.MessageWithParts
	}
	sendResultMsg struct {
		sessionID string
		err       error
	}
	errMsg  struct{ err error }
	tickMsg struct{}
)

// New creates a new Model using the given OpenCodeClients.
func New(clients ...client.OpenCodeClient) Model {
	return Model{
		clients:         clients,
		statuses:        make(map[string]client.SessionStatus),
		messages:        make(map[string][]client.MessageWithParts),
		agentNames:      make(map[string]string),
		focusedPane:     focusSidebar,
		swapMark:        -1,
		refreshInterval: parseRefreshInterval(),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.clients)*2+1)
	for i, oc := range m.clients {
		idx := i
		cmds = append(cmds, fetchSessions(oc, idx))
		cmds = append(cmds, startSSE(oc, idx))
	}
	cmds = append(cmds, tick(m.refreshInterval))
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	case refreshMsg:
		m = m.mergeRefresh(msg)
		m = m.rebuildTree()
		return m, m.fetchOpenPaneMessages()
	case sseMsg:
		return m.handleSSE(msg.clientIdx, msg.event)
	case messagesMsg:
		m.messages[msg.sessionID] = msg.messages
		if name := extractAgentName(msg.messages); name != "" {
			if m.agentNames[msg.sessionID] != name {
				m.agentNames[msg.sessionID] = name
				m = m.rebuildTree()
			}
		}
	case sendResultMsg:
		if msg.err != nil {
			m.err = msg.err
		}
	case tickMsg:
		cmds := make([]tea.Cmd, 0, len(m.clients)+1)
		for i, oc := range m.clients {
			cmds = append(cmds, fetchSessions(oc, i))
		}
		cmds = append(cmds, tick(m.refreshInterval))
		return m, tea.Batch(cmds...)
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

	sidebarW := m.width * sidebarWidthPercent / 100
	if sidebarW < 20 {
		sidebarW = 20
	}
	panesW := m.width - sidebarW - 1 // -1 for gap
	contentH := m.height - helpBarHeight

	// Render sidebar
	treeContent := renderTree(m.tree, m.selectedSidebarID(), sidebarW-layoutPadding)
	sidebarPanel := panelBox("Sessions", treeContent, sidebarW, contentH, m.focusedPane == focusSidebar)

	// Render panes area
	var panesArea string
	if len(m.panes) == 0 {
		emptyContent := styleDim.Render("  Press Enter on a session to open it here")
		panesArea = panelBox("Panes", emptyContent, panesW, contentH, false)
	} else {
		panesArea = m.renderPanesArea(panesW, contentH)
	}

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebarPanel, panesArea)

	help := styleHelp.Render(helpText)
	if m.err != nil {
		help = lipgloss.NewStyle().Foreground(colorRed).PaddingLeft(1).
			Render(fmt.Sprintf("Error: %v", m.err))
	}

	v.Content = lipgloss.JoinVertical(lipgloss.Left, topRow, help)
	return v
}

func (m Model) renderPanesArea(totalW, totalH int) string {
	rects := calcLayout(len(m.panes), totalW, totalH)

	// Group by rows to join horizontally, then vertically
	type row struct {
		y     int
		panes []string
	}
	rows := make(map[int]*row)
	var rowKeys []int

	for i, rect := range rects {
		p := m.panes[i]
		sess := m.findSession(p.sessionID)
		status := ""
		if st, ok := m.statuses[p.sessionID]; ok {
			status = st.Type
		}
		msgs := m.messages[p.sessionID]
		focused := m.focusedPane == i
		swapMarked := m.swapMark == i

		rendered := renderPane(p, sess, status, msgs, rect.W, rect.H, focused, swapMarked)

		if _, ok := rows[rect.Y]; !ok {
			rows[rect.Y] = &row{y: rect.Y}
			rowKeys = append(rowKeys, rect.Y)
		}
		rows[rect.Y].panes = append(rows[rect.Y].panes, rendered)
	}

	sort.Ints(rowKeys)
	var renderedRows []string
	for _, y := range rowKeys {
		r := rows[y]
		renderedRows = append(renderedRows, lipgloss.JoinHorizontal(lipgloss.Top, r.panes...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, renderedRows...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input mode first
	if m.focusedPane >= 0 && m.focusedPane < len(m.panes) && m.panes[m.focusedPane].inputMode {
		return m.handleInputKey(msg)
	}

	switch msg.String() {
	case "q":
		for _, oc := range m.clients {
			oc.StopEvents()
		}
		return m, tea.Quit
	case "esc":
		// If swap mark active, cancel it
		if m.swapMark >= 0 {
			m.swapMark = -1
			return m, nil
		}
		for _, oc := range m.clients {
			oc.StopEvents()
		}
		return m, tea.Quit

	case "tab":
		m = m.cycleFocus(1)
	case "shift+tab":
		m = m.cycleFocus(-1)

	case "j", "down":
		if m.focusedPane == focusSidebar {
			m.sidebarCursor++
			m = m.clampSidebarCursor()
		} else if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].followMode = false
			m.panes[m.focusedPane].scroll++
		}
	case "k", "up":
		if m.focusedPane == focusSidebar {
			m.sidebarCursor--
			m = m.clampSidebarCursor()
		} else if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].followMode = false
			if m.panes[m.focusedPane].scroll > 0 {
				m.panes[m.focusedPane].scroll--
			}
		}

	case "enter":
		if m.focusedPane == focusSidebar {
			return m.openSelectedSession()
		}
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].expandTools = !m.panes[m.focusedPane].expandTools
		}

	case "x":
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes = removePane(m.panes, m.focusedPane)
			if m.swapMark == m.focusedPane {
				m.swapMark = -1
			} else if m.swapMark > m.focusedPane {
				m.swapMark--
			}
			if len(m.panes) == 0 {
				m.focusedPane = focusSidebar
			} else if m.focusedPane >= len(m.panes) {
				m.focusedPane = len(m.panes) - 1
			}
		}

	case "i":
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].inputMode = true
			m.panes[m.focusedPane].input = newTextInput()
		}

	case "m":
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			if m.swapMark < 0 {
				m.swapMark = m.focusedPane
			} else {
				m.panes = swapPanes(m.panes, m.swapMark, m.focusedPane)
				m.swapMark = -1
			}
		}

	case "G":
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].followMode = true
			m.panes[m.focusedPane].scroll = maxScrollVal
		}

	case "f":
		m.showAll = !m.showAll
		m = m.rebuildTree()

	case "a":
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			oc := m.clients[p.clientIdx]
			return m, abortSession(oc, p.sessionID)
		}
		if m.focusedPane == focusSidebar {
			if id := m.selectedSidebarID(); id != "" {
				idx := findNodeClientIdx(m.tree, id)
				if idx >= 0 && idx < len(m.clients) {
					return m, abortSession(m.clients[idx], id)
				}
			}
		}

	case "r":
		cmds := make([]tea.Cmd, len(m.clients))
		for i, oc := range m.clients {
			cmds[i] = fetchSessions(oc, i)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := &m.panes[m.focusedPane]

	switch msg.String() {
	case "esc":
		p.inputMode = false
		p.input = p.input.clear()
		return m, nil
	case "enter":
		text := p.input.value()
		if text == "" {
			return m, nil
		}
		sessionID := p.sessionID
		oc := m.clients[p.clientIdx]
		p.inputMode = false
		p.input = p.input.clear()
		p.followMode = true
		p.scroll = maxScrollVal
		return m, sendMessage(oc, sessionID, text)
	case "backspace":
		p.input = p.input.backspace()
	case "delete":
		p.input = p.input.delete()
	case "left":
		p.input = p.input.moveLeft()
	case "right":
		p.input = p.input.moveRight()
	case "home":
		p.input = p.input.moveHome()
	case "end":
		p.input = p.input.moveEnd()
	default:
		// Insert printable characters
		key := msg.String()
		if len(key) == 1 {
			p.input = p.input.insert(rune(key[0]))
		} else if key == "space" {
			p.input = p.input.insert(' ')
		}
	}
	return m, nil
}

func (m Model) openSelectedSession() (Model, tea.Cmd) {
	id := m.selectedSidebarID()
	if id == "" {
		return m, nil
	}
	clientIdx := findNodeClientIdx(m.tree, id)
	if clientIdx < 0 {
		clientIdx = 0
	}
	var idx int
	m.panes, idx = addPane(m.panes, id, clientIdx)
	m.focusedPane = idx

	oc := m.clients[clientIdx]
	return m, fetchMessages(oc, id)
}

func (m Model) cycleFocus(dir int) Model {
	totalSlots := 1 + len(m.panes) // sidebar + panes
	if totalSlots <= 1 {
		return m
	}

	// Current position: sidebar=-1 maps to 0, pane i maps to i+1
	current := m.focusedPane + 1 // now 0 = sidebar, 1..n = panes
	current = (current + dir + totalSlots) % totalSlots
	m.focusedPane = current - 1 // back to -1 = sidebar, 0..n-1 = panes
	return m
}

func (m Model) handleSSE(clientIdx int, event client.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case client.EventSessionStatus:
		var e client.SessionStatusEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			m.statuses[e.SessionID] = e.Status
			m = m.rebuildTree()
		}
	case client.EventSessionCreated:
		var e client.SessionEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			m.allSessions = append(m.allSessions, sessionEntry{session: e.Info, clientIdx: clientIdx})
			m = m.rebuildTree()
			return m, tea.Batch(
				listenSSE(m.clients[clientIdx], clientIdx),
				fetchMessages(m.clients[clientIdx], e.Info.ID),
			)
		}
	case client.EventSessionUpdated:
		var e client.SessionEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			for i := range m.allSessions {
				if m.allSessions[i].session.ID == e.Info.ID {
					m.allSessions[i].session = e.Info
					break
				}
			}
			m = m.rebuildTree()
		}
	case client.EventMessagePartUpdate:
		var e client.MessagePartEvent
		if json.Unmarshal(event.Properties, &e) == nil {
			sid := e.Part.SessionID
			// Update follow mode for any open pane showing this session
			for i := range m.panes {
				if m.panes[i].sessionID == sid && m.panes[i].followMode {
					m.panes[i].scroll = maxScrollVal
				}
			}
			if findPaneBySession(m.panes, sid) >= 0 {
				return m, tea.Batch(
					listenSSE(m.clients[clientIdx], clientIdx),
					fetchMessages(m.clients[clientIdx], sid),
				)
			}
		}
	case client.EventTodoUpdated:
		// handled by refresh
	}
	return m, listenSSE(m.clients[clientIdx], clientIdx)
}

func (m Model) mergeRefresh(msg refreshMsg) Model {
	// Remove old sessions from this client
	var kept []sessionEntry
	for _, e := range m.allSessions {
		if e.clientIdx != msg.clientIdx {
			kept = append(kept, e)
		}
	}
	// Add new sessions from this client
	for _, s := range msg.sessions {
		kept = append(kept, sessionEntry{session: s, clientIdx: msg.clientIdx})
	}
	m.allSessions = kept

	// Merge statuses
	for k, v := range msg.statuses {
		m.statuses[k] = v
	}
	return m
}

func (m Model) rebuildTree() Model {
	m.tree = buildTreeMulti(m.allSessions, m.statuses, m.agentNames)
	if !m.showAll {
		m.tree = pruneIdle(m.tree)
	}
	m.flatIDs = flattenTree(m.tree)
	m = m.clampSidebarCursor()
	return m
}

func pruneIdle(nodes []*treeNode) []*treeNode {
	var result []*treeNode
	for _, node := range nodes {
		node.children = pruneIdle(node.children)
		if node.status == statusBusy || len(node.children) > 0 {
			result = append(result, node)
		}
	}
	return result
}

func (m Model) selectedSidebarID() string {
	if len(m.flatIDs) == 0 {
		return ""
	}
	if m.sidebarCursor >= len(m.flatIDs) {
		return m.flatIDs[len(m.flatIDs)-1]
	}
	return m.flatIDs[m.sidebarCursor]
}

func (m Model) clampSidebarCursor() Model {
	max := len(m.flatIDs) - 1
	if max < 0 {
		max = 0
	}
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	if m.sidebarCursor > max {
		m.sidebarCursor = max
	}
	return m
}

func (m Model) findSession(id string) *client.Session {
	for i := range m.allSessions {
		if m.allSessions[i].session.ID == id {
			return &m.allSessions[i].session
		}
	}
	return nil
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

func (m Model) fetchOpenPaneMessages() tea.Cmd {
	if len(m.panes) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(m.panes))
	for _, p := range m.panes {
		if p.clientIdx < len(m.clients) {
			cmds = append(cmds, fetchMessages(m.clients[p.clientIdx], p.sessionID))
		}
	}
	return tea.Batch(cmds...)
}

// --- Commands ---

func fetchSessions(oc client.OpenCodeClient, clientIdx int) tea.Cmd {
	return func() tea.Msg {
		sessions, err := oc.Sessions()
		if err != nil {
			return errMsg{err}
		}
		statuses, err := oc.SessionStatuses()
		if err != nil {
			return errMsg{err}
		}
		return refreshMsg{clientIdx: clientIdx, sessions: sessions, statuses: statuses}
	}
}

func startSSE(oc client.OpenCodeClient, clientIdx int) tea.Cmd {
	return func() tea.Msg {
		oc.StartEvents()
		event, ok := <-oc.Events()
		if !ok {
			return nil
		}
		return sseMsg{clientIdx: clientIdx, event: event}
	}
}

func listenSSE(oc client.OpenCodeClient, clientIdx int) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-oc.Events()
		if !ok {
			return nil
		}
		return sseMsg{clientIdx: clientIdx, event: event}
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

func sendMessage(oc client.OpenCodeClient, sessionID, text string) tea.Cmd {
	return func() tea.Msg {
		err := oc.SendMessage(sessionID, text)
		return sendResultMsg{sessionID: sessionID, err: err}
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

func extractAgentName(messages []client.MessageWithParts) string {
	for _, msg := range messages {
		if msg.Info.Agent != "" {
			return msg.Info.Agent
		}
	}
	return ""
}

func parseRefreshInterval() time.Duration {
	if v := os.Getenv(envRefreshInterval); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultRefreshInterval
}
