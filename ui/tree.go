package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/tkozakas/agent-monitor/client"
)

const (
	connectorBranch = "├─"
	connectorLast   = "└─"
	connectorRoot   = "▸"
	connectorPipe   = "│  "
	cursorIndicator = "❯ "
	indentSpaces    = "  "
	indentChild     = "   "

	maxTitleLen = 30
	titleIDLen  = 8
)

type treeNode struct {
	session   client.Session
	status    string
	agent     string
	clientIdx int
	children  []*treeNode
}

// sessionEntry pairs a session with the client index it came from.
type sessionEntry struct {
	session   client.Session
	clientIdx int
}

func buildTree(sessions []client.Session, statuses map[string]client.SessionStatus, agentNames map[string]string) []*treeNode {
	return buildTreeMulti(toEntries(sessions, 0), statuses, agentNames)
}

func toEntries(sessions []client.Session, clientIdx int) []sessionEntry {
	entries := make([]sessionEntry, len(sessions))
	for i, s := range sessions {
		entries[i] = sessionEntry{session: s, clientIdx: clientIdx}
	}
	return entries
}

func buildTreeMulti(entries []sessionEntry, statuses map[string]client.SessionStatus, agentNames map[string]string) []*treeNode {
	byID := make(map[string]*treeNode, len(entries))
	var roots []*treeNode

	for _, e := range entries {
		s := e.session
		status := statusIdle
		if st, ok := statuses[s.ID]; ok {
			status = st.Type
		}
		var agent string
		if agentNames != nil {
			agent = agentNames[s.ID]
		}
		byID[s.ID] = &treeNode{session: s, status: status, agent: agent, clientIdx: e.clientIdx}
	}

	for _, node := range byID {
		if node.session.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		if parent, ok := byID[*node.session.ParentID]; ok {
			parent.children = append(parent.children, node)
		} else {
			roots = append(roots, node)
		}
	}

	return roots
}

func renderTree(nodes []*treeNode, selected string, width int) string {
	var b strings.Builder
	for _, node := range nodes {
		renderNode(&b, node, selected, "", true, width)
	}
	return b.String()
}

func renderNode(b *strings.Builder, node *treeNode, selected, prefix string, last bool, width int) {
	connector := connectorBranch
	if last {
		connector = connectorLast
	}
	if prefix == "" {
		connector = connectorRoot
	}

	icon := statusIcon(node.status)
	clr := statusColor(node.status)
	name := nodeLabel(node)

	line := fmt.Sprintf("%s%s %s %s %s",
		prefix, connector,
		lipgloss.NewStyle().Foreground(clr).Render(icon),
		lipgloss.NewStyle().Bold(true).Render(name),
		styleDim.Render(node.status),
	)

	if node.session.ID == selected {
		line = styleSelected.Render(cursorIndicator) + line
	} else {
		line = indentSpaces + line
	}

	b.WriteString(line)
	b.WriteString("\n")

	if topic := nodeTopic(node); topic != "" {
		topicPrefix := indentSpaces
		if node.session.ID == selected {
			topicPrefix = indentSpaces
		}
		if prefix == "" {
			topicPrefix += indentSpaces + indentSpaces
		} else {
			topicPrefix += prefix
			if last {
				topicPrefix += indentChild
			} else {
				topicPrefix += connectorPipe
			}
		}
		b.WriteString(topicPrefix + styleDim.Render(topic) + "\n")
	}

	childPrefix := prefix
	if prefix == "" {
		childPrefix = indentSpaces
	} else if last {
		childPrefix += indentChild
	} else {
		childPrefix += connectorPipe
	}

	for i, child := range node.children {
		isLast := i == len(node.children)-1
		renderNode(b, child, selected, childPrefix, isLast, width)
	}
}

func flattenTree(nodes []*treeNode) []string {
	var ids []string
	for _, node := range nodes {
		ids = append(ids, node.session.ID)
		ids = append(ids, flattenTree(node.children)...)
	}
	return ids
}

func nodeLabel(node *treeNode) string {
	name := node.agent
	if name == "" {
		name = node.session.Title
	}
	if name == "" && len(node.session.ID) >= titleIDLen {
		name = node.session.ID[:titleIDLen]
	}
	if len(name) > maxTitleLen {
		name = name[:maxTitleLen-3] + "..."
	}
	return name
}

func nodeTopic(node *treeNode) string {
	if node.agent == "" {
		return ""
	}
	title := node.session.Title
	if title == "" {
		return ""
	}
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}
	return title
}

// findNodeClientIdx looks up the clientIdx for a session ID in the tree.
func findNodeClientIdx(nodes []*treeNode, sessionID string) int {
	for _, node := range nodes {
		if node.session.ID == sessionID {
			return node.clientIdx
		}
		if idx := findNodeClientIdx(node.children, sessionID); idx >= 0 {
			return idx
		}
	}
	return -1
}
