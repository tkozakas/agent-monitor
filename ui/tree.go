package ui

import (
	"fmt"
	"sort"
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
	agent    string
	children []*treeNode
}

func buildTree(sessions []client.Session, statuses map[string]client.SessionStatus, agentNames map[string]string) []*treeNode {
	byID := make(map[string]*treeNode, len(sessions))
	var roots []*treeNode

	for _, s := range sessions {
		status := statusIdle
		if st, ok := statuses[s.ID]; ok {
			status = st.Type
		}
		var agent string
		if agentNames != nil {
			agent = agentNames[s.ID]
		}
		byID[s.ID] = &treeNode{session: s, status: status, agent: agent}
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

const graphNodeGap = 3

type layoutNode struct {
	node       *treeNode
	centerX    int
	totalWidth int
	children   []*layoutNode
}

func renderGraphTree(roots []*treeNode, selected string, width int) string {
	if len(roots) == 0 {
		return styleDim.Render("No agents")
	}

	root := roots[0]
	ln := buildGraphLayout(root, 0)

	offset := (width - ln.totalWidth) / 2
	if offset < 0 {
		offset = 0
	}
	shiftGraphLayout(ln, offset)

	var levels [][]*layoutNode
	collectGraphLevels(ln, 0, &levels)

	var lines []string
	for i, level := range levels {
		lines = append(lines, renderGraphNodeLine(level, selected, width))
		if i < len(levels)-1 {
			lines = append(lines, renderGraphConnectors(level, width)...)
		}
	}
	return strings.Join(lines, "\n")
}

func buildGraphLayout(node *treeNode, startX int) *layoutNode {
	ln := &layoutNode{node: node}
	cellW := graphCellWidth(node)

	if len(node.children) == 0 {
		ln.totalWidth = cellW
		ln.centerX = startX + cellW/2
		return ln
	}

	childrenW := 0
	for i, child := range node.children {
		if i > 0 {
			childrenW += graphNodeGap
		}
		childrenW += graphSubtreeWidth(child)
	}

	ln.totalWidth = childrenW
	if cellW > childrenW {
		ln.totalWidth = cellW
	}

	childStart := startX
	if childrenW < ln.totalWidth {
		childStart += (ln.totalWidth - childrenW) / 2
	}

	x := childStart
	for i, child := range node.children {
		if i > 0 {
			x += graphNodeGap
		}
		cln := buildGraphLayout(child, x)
		ln.children = append(ln.children, cln)
		x += graphSubtreeWidth(child)
	}

	ln.centerX = startX + ln.totalWidth/2

	return ln
}

func graphSubtreeWidth(node *treeNode) int {
	cellW := graphCellWidth(node)
	if len(node.children) == 0 {
		return cellW
	}
	childrenW := 0
	for i, child := range node.children {
		if i > 0 {
			childrenW += graphNodeGap
		}
		childrenW += graphSubtreeWidth(child)
	}
	if cellW > childrenW {
		return cellW
	}
	return childrenW
}

func graphCellWidth(node *treeNode) int {
	return 2 + len(nodeLabel(node)) + 1 + len(node.status)
}

func shiftGraphLayout(ln *layoutNode, offset int) {
	ln.centerX += offset
	for _, child := range ln.children {
		shiftGraphLayout(child, offset)
	}
}

func collectGraphLevels(ln *layoutNode, level int, levels *[][]*layoutNode) {
	for len(*levels) <= level {
		*levels = append(*levels, nil)
	}
	(*levels)[level] = append((*levels)[level], ln)
	for _, child := range ln.children {
		collectGraphLevels(child, level+1, levels)
	}
}

func renderGraphNodeLine(nodes []*layoutNode, selected string, width int) string {
	sorted := make([]*layoutNode, len(nodes))
	copy(sorted, nodes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].centerX < sorted[j].centerX
	})

	var b strings.Builder
	cursor := 0

	for _, ln := range sorted {
		icon := statusIcon(ln.node.status)
		clr := statusColor(ln.node.status)
		name := nodeLabel(ln.node)
		status := ln.node.status

		cellVis := 2 + len(name) + 1 + len(status)
		startX := ln.centerX - cellVis/2
		if startX < cursor {
			startX = cursor
		}

		for cursor < startX {
			b.WriteRune(' ')
			cursor++
		}

		isSelected := ln.node.session.ID == selected
		if isSelected {
			b.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Render(icon))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(clr).Render(icon))
		}
		b.WriteRune(' ')

		if isSelected {
			b.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(name))
		} else {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render(name))
		}
		b.WriteRune(' ')
		b.WriteString(styleDim.Render(status))

		cursor = startX + cellVis
	}

	return b.String()
}

func renderGraphConnectors(parents []*layoutNode, width int) []string {
	maxX := 0
	for _, p := range parents {
		if p.centerX > maxX {
			maxX = p.centerX
		}
		for _, c := range p.children {
			if c.centerX > maxX {
				maxX = c.centerX
			}
		}
	}

	lineLen := maxX + 1
	line1 := make([]rune, lineLen)
	line2 := make([]rune, lineLen)
	for i := range line1 {
		line1[i] = ' '
		line2[i] = ' '
	}

	for _, p := range parents {
		if len(p.children) == 0 {
			continue
		}

		if len(p.children) == 1 {
			cx := p.centerX
			if cx < lineLen {
				line1[cx] = '│'
				line2[cx] = '│'
			}
			continue
		}

		if p.centerX < lineLen {
			line1[p.centerX] = '│'
		}

		leftmost := p.children[0].centerX
		rightmost := p.children[len(p.children)-1].centerX

		for x := leftmost; x <= rightmost && x < lineLen; x++ {
			line2[x] = '─'
		}

		if leftmost < lineLen {
			line2[leftmost] = '┌'
		}
		if rightmost < lineLen {
			line2[rightmost] = '┐'
		}

		if p.centerX >= leftmost && p.centerX <= rightmost && p.centerX < lineLen {
			line2[p.centerX] = '┴'
		}

		for _, c := range p.children[1 : len(p.children)-1] {
			if c.centerX < lineLen {
				line2[c.centerX] = '┬'
			}
		}
	}

	return []string{string(line1), string(line2)}
}
