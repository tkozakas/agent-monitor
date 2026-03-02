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
	session  client.Session
	status   string
	children []*treeNode
}

func buildTree(sessions []client.Session, statuses map[string]client.SessionStatus) []*treeNode {
	byID := make(map[string]*treeNode, len(sessions))
	var roots []*treeNode

	for _, s := range sessions {
		status := statusIdle
		if st, ok := statuses[s.ID]; ok {
			status = st.Type
		}
		byID[s.ID] = &treeNode{session: s, status: status}
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

	title := node.session.Title
	if title == "" && len(node.session.ID) >= titleIDLen {
		title = node.session.ID[:titleIDLen]
	}
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	line := fmt.Sprintf("%s%s %s %s",
		prefix, connector,
		lipgloss.NewStyle().Foreground(clr).Render(icon),
		title,
	)
	line += " " + lipgloss.NewStyle().Foreground(clr).Render(fmt.Sprintf("[%s]", node.status))

	if node.session.ID == selected {
		line = styleSelected.Render(cursorIndicator) + line
	} else {
		line = indentSpaces + line
	}

	b.WriteString(line)
	b.WriteString("\n")

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
