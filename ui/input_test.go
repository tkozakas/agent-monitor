package ui

import (
	"testing"
)

func TestTextInputInsert(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('h')
	ti = ti.insert('i')
	if ti.value() != "hi" {
		t.Errorf("expected 'hi', got %q", ti.value())
	}
	if ti.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", ti.cursor)
	}
}

func TestTextInputBackspace(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('a')
	ti = ti.insert('b')
	ti = ti.insert('c')
	ti = ti.backspace()
	if ti.value() != "ab" {
		t.Errorf("expected 'ab', got %q", ti.value())
	}
}

func TestTextInputBackspaceEmpty(t *testing.T) {
	ti := newTextInput()
	ti = ti.backspace()
	if ti.value() != "" {
		t.Errorf("expected empty, got %q", ti.value())
	}
}

func TestTextInputDelete(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('a')
	ti = ti.insert('b')
	ti = ti.insert('c')
	ti = ti.moveHome()
	ti = ti.delete()
	if ti.value() != "bc" {
		t.Errorf("expected 'bc', got %q", ti.value())
	}
}

func TestTextInputMovement(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('a')
	ti = ti.insert('b')
	ti = ti.insert('c')
	ti = ti.moveHome()
	if ti.cursor != 0 {
		t.Errorf("expected cursor 0 after home, got %d", ti.cursor)
	}
	ti = ti.moveEnd()
	if ti.cursor != 3 {
		t.Errorf("expected cursor 3 after end, got %d", ti.cursor)
	}
	ti = ti.moveLeft()
	if ti.cursor != 2 {
		t.Errorf("expected cursor 2 after left, got %d", ti.cursor)
	}
	ti = ti.moveRight()
	if ti.cursor != 3 {
		t.Errorf("expected cursor 3 after right, got %d", ti.cursor)
	}
}

func TestTextInputClear(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('x')
	ti = ti.insert('y')
	ti = ti.clear()
	if ti.value() != "" {
		t.Errorf("expected empty after clear, got %q", ti.value())
	}
	if ti.cursor != 0 {
		t.Errorf("expected cursor 0 after clear, got %d", ti.cursor)
	}
}

func TestTextInputRender(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('h')
	ti = ti.insert('e')
	ti = ti.insert('l')
	ti = ti.insert('l')
	ti = ti.insert('o')
	result := ti.render(40)
	if result == "" {
		t.Error("expected non-empty render output")
	}
}

func TestTextInputMoveLeftBound(t *testing.T) {
	ti := newTextInput()
	ti = ti.moveLeft()
	if ti.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", ti.cursor)
	}
}

func TestTextInputMoveRightBound(t *testing.T) {
	ti := newTextInput()
	ti = ti.moveRight()
	if ti.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", ti.cursor)
	}
}

func TestTextInputInsertMiddle(t *testing.T) {
	ti := newTextInput()
	ti = ti.insert('a')
	ti = ti.insert('c')
	ti = ti.moveLeft()
	ti = ti.insert('b')
	if ti.value() != "abc" {
		t.Errorf("expected 'abc', got %q", ti.value())
	}
}
