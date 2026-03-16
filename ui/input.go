package ui

import "strings"

// textInput is a minimal single-line text input component.
type textInput struct {
	text   []rune
	cursor int
}

func newTextInput() textInput {
	return textInput{}
}

func (t textInput) insert(ch rune) textInput {
	before := t.text[:t.cursor]
	after := make([]rune, len(t.text[t.cursor:]))
	copy(after, t.text[t.cursor:])
	t.text = append(append(before, ch), after...)
	t.cursor++
	return t
}

func (t textInput) backspace() textInput {
	if t.cursor == 0 || len(t.text) == 0 {
		return t
	}
	before := make([]rune, t.cursor-1)
	copy(before, t.text[:t.cursor-1])
	after := make([]rune, len(t.text[t.cursor:]))
	copy(after, t.text[t.cursor:])
	t.text = append(before, after...)
	t.cursor--
	return t
}

func (t textInput) delete() textInput {
	if t.cursor >= len(t.text) {
		return t
	}
	before := make([]rune, t.cursor)
	copy(before, t.text[:t.cursor])
	after := make([]rune, len(t.text[t.cursor+1:]))
	copy(after, t.text[t.cursor+1:])
	t.text = append(before, after...)
	return t
}

func (t textInput) moveLeft() textInput {
	if t.cursor > 0 {
		t.cursor--
	}
	return t
}

func (t textInput) moveRight() textInput {
	if t.cursor < len(t.text) {
		t.cursor++
	}
	return t
}

func (t textInput) moveHome() textInput {
	t.cursor = 0
	return t
}

func (t textInput) moveEnd() textInput {
	t.cursor = len(t.text)
	return t
}

func (t textInput) value() string {
	return string(t.text)
}

func (t textInput) clear() textInput {
	t.text = nil
	t.cursor = 0
	return t
}

func (t textInput) render(width int) string {
	if width <= 4 {
		return "> "
	}

	prompt := "> "
	available := width - len(prompt) - 1 // -1 for cursor
	text := string(t.text)
	cur := t.cursor

	// Scroll if cursor is past visible area
	offset := 0
	if cur > available {
		offset = cur - available
	}

	visible := text
	if offset > 0 && offset < len([]rune(text)) {
		visible = string([]rune(text)[offset:])
	}
	if len([]rune(visible)) > available+1 {
		visible = string([]rune(visible)[:available+1])
	}

	// Insert cursor character
	visibleRunes := []rune(visible)
	curPos := cur - offset
	if curPos < 0 {
		curPos = 0
	}
	if curPos > len(visibleRunes) {
		curPos = len(visibleRunes)
	}

	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString(string(visibleRunes[:curPos]))
	b.WriteString("█")
	if curPos < len(visibleRunes) {
		b.WriteString(string(visibleRunes[curPos:]))
	}

	return b.String()
}
