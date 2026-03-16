package ui

import (
	"testing"
)

func TestCalcLayoutSingle(t *testing.T) {
	rects := calcLayout(1, 100, 50)
	if len(rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(rects))
	}
	r := rects[0]
	if r.X != 0 || r.Y != 0 || r.W != 100 || r.H != 50 {
		t.Errorf("expected {0,0,100,50}, got {%d,%d,%d,%d}", r.X, r.Y, r.W, r.H)
	}
}

func TestCalcLayoutTwo(t *testing.T) {
	rects := calcLayout(2, 100, 50)
	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	// Side by side: 2 cols, 1 row
	if rects[0].W+rects[1].W != 100 {
		t.Errorf("expected widths to sum to 100, got %d+%d", rects[0].W, rects[1].W)
	}
	if rects[0].H != 50 || rects[1].H != 50 {
		t.Errorf("expected height 50, got %d and %d", rects[0].H, rects[1].H)
	}
}

func TestCalcLayoutFour(t *testing.T) {
	rects := calcLayout(4, 100, 100)
	if len(rects) != 4 {
		t.Fatalf("expected 4 rects, got %d", len(rects))
	}
	// 2x2 grid
	if rects[0].W != 50 || rects[0].H != 50 {
		t.Errorf("expected 50x50 for first pane, got %dx%d", rects[0].W, rects[0].H)
	}
}

func TestCalcLayoutThree(t *testing.T) {
	rects := calcLayout(3, 100, 100)
	if len(rects) != 3 {
		t.Fatalf("expected 3 rects, got %d", len(rects))
	}
	// 2 cols, 2 rows: first row has 2, second row has 1 (full width)
	if rects[2].W != 100 {
		t.Errorf("expected last pane full width 100, got %d", rects[2].W)
	}
}

func TestCalcLayoutZero(t *testing.T) {
	rects := calcLayout(0, 100, 100)
	if len(rects) != 0 {
		t.Errorf("expected 0 rects, got %d", len(rects))
	}
}

func TestSwapPanes(t *testing.T) {
	panes := []Pane{
		{sessionID: "a"},
		{sessionID: "b"},
		{sessionID: "c"},
	}
	panes = swapPanes(panes, 0, 2)
	if panes[0].sessionID != "c" || panes[2].sessionID != "a" {
		t.Errorf("swap failed: got %s and %s", panes[0].sessionID, panes[2].sessionID)
	}
}

func TestSwapPanesInvalidIndex(t *testing.T) {
	panes := []Pane{{sessionID: "a"}}
	result := swapPanes(panes, 0, 5)
	if result[0].sessionID != "a" {
		t.Error("expected no change for invalid index")
	}
}

func TestSwapPanesSameIndex(t *testing.T) {
	panes := []Pane{{sessionID: "a"}, {sessionID: "b"}}
	panes = swapPanes(panes, 1, 1)
	if panes[1].sessionID != "b" {
		t.Error("expected no change for same index")
	}
}

func TestAddPane(t *testing.T) {
	var panes []Pane
	panes, idx := addPane(panes, "s1", 0)
	if len(panes) != 1 || idx != 0 {
		t.Errorf("expected 1 pane at idx 0, got %d panes at idx %d", len(panes), idx)
	}
	// Adding same session should not duplicate
	panes, idx = addPane(panes, "s1", 0)
	if len(panes) != 1 || idx != 0 {
		t.Errorf("expected still 1 pane, got %d", len(panes))
	}
	// Adding different session
	panes, idx = addPane(panes, "s2", 1)
	if len(panes) != 2 || idx != 1 {
		t.Errorf("expected 2 panes at idx 1, got %d at %d", len(panes), idx)
	}
}

func TestRemovePane(t *testing.T) {
	panes := []Pane{
		{sessionID: "a"},
		{sessionID: "b"},
		{sessionID: "c"},
	}
	panes = removePane(panes, 1)
	if len(panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(panes))
	}
	if panes[0].sessionID != "a" || panes[1].sessionID != "c" {
		t.Errorf("expected [a, c], got [%s, %s]", panes[0].sessionID, panes[1].sessionID)
	}
}

func TestRemovePaneInvalid(t *testing.T) {
	panes := []Pane{{sessionID: "a"}}
	result := removePane(panes, -1)
	if len(result) != 1 {
		t.Error("expected no change for invalid index")
	}
	result = removePane(panes, 5)
	if len(result) != 1 {
		t.Error("expected no change for out of bounds index")
	}
}

func TestFindPaneBySession(t *testing.T) {
	panes := []Pane{
		{sessionID: "a"},
		{sessionID: "b"},
	}
	if idx := findPaneBySession(panes, "b"); idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
	if idx := findPaneBySession(panes, "c"); idx != -1 {
		t.Errorf("expected -1 for missing, got %d", idx)
	}
}
