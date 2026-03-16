package ui

import "math"

// Pane represents an open session pane in the right-side tiled view.
type Pane struct {
	sessionID   string
	clientIdx   int
	scroll      int
	followMode  bool
	expandTools bool
	inputMode   bool
	input       textInput
}

// PaneRect holds the calculated position and size for a pane.
type PaneRect struct {
	X, Y, W, H int
}

func newPane(sessionID string, clientIdx int) Pane {
	return Pane{
		sessionID:  sessionID,
		clientIdx:  clientIdx,
		followMode: true,
		input:      newTextInput(),
	}
}

// calcLayout computes an auto-grid layout for n panes in the given area.
// Layout strategy: cols = ceil(sqrt(n)), rows = ceil(n/cols).
// Last row may have fewer panes that expand to fill the width.
func calcLayout(n, totalW, totalH int) []PaneRect {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []PaneRect{{X: 0, Y: 0, W: totalW, H: totalH}}
	}

	cols := int(math.Ceil(math.Sqrt(float64(n))))
	rows := int(math.Ceil(float64(n) / float64(cols)))

	rects := make([]PaneRect, 0, n)
	rowH := totalH / rows

	idx := 0
	for r := 0; r < rows; r++ {
		remaining := n - idx
		rowCols := cols
		if remaining < cols {
			rowCols = remaining
		}

		colW := totalW / rowCols
		y := r * rowH

		// Last row gets remaining height
		h := rowH
		if r == rows-1 {
			h = totalH - y
		}

		for c := 0; c < rowCols; c++ {
			x := c * colW
			w := colW
			// Last column in row gets remaining width
			if c == rowCols-1 {
				w = totalW - x
			}
			rects = append(rects, PaneRect{X: x, Y: y, W: w, H: h})
			idx++
		}
	}

	return rects
}

// swapPanes swaps two panes by index.
func swapPanes(panes []Pane, a, b int) []Pane {
	if a < 0 || b < 0 || a >= len(panes) || b >= len(panes) || a == b {
		return panes
	}
	panes[a], panes[b] = panes[b], panes[a]
	return panes
}

// addPane adds a new pane if the session is not already open.
// Returns the panes slice and the index of the new or existing pane.
func addPane(panes []Pane, sessionID string, clientIdx int) ([]Pane, int) {
	for i, p := range panes {
		if p.sessionID == sessionID {
			return panes, i
		}
	}
	panes = append(panes, newPane(sessionID, clientIdx))
	return panes, len(panes) - 1
}

// removePane removes a pane by index and returns the updated slice.
func removePane(panes []Pane, idx int) []Pane {
	if idx < 0 || idx >= len(panes) {
		return panes
	}
	return append(panes[:idx], panes[idx+1:]...)
}

// findPaneBySession returns the index of a pane for the given session, or -1.
func findPaneBySession(panes []Pane, sessionID string) int {
	for i, p := range panes {
		if p.sessionID == sessionID {
			return i
		}
	}
	return -1
}
