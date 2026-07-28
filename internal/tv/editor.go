package tv

import (
	"strconv"

	"github.com/gdamore/tcell/v2"

	"sidekick/internal/theme"
)

// Editor is a minimal Notepad-style text buffer: typing, cursor movement,
// line insert/join, and scrolling. No undo, clipboard, word-wrap or
// save/load. It implements WindowContent.
type Editor struct {
	lines                []string
	cursorRow, cursorCol int
	topLine, hOffset     int
	tabWidth             int

	lastW, lastH int // content rect size from the most recent Draw
}

func NewEditor() *Editor {
	return &Editor{lines: []string{""}, tabWidth: 8}
}

// Draw renders the ruler on the content's first row, then text lines
// below it, scrolled to keep the cursor visible.
func (e *Editor) Draw(screen tcell.Screen, x, y, w, h int) {
	e.lastW, e.lastH = w, h
	if w <= 0 || h <= 0 {
		return
	}
	visibleRows := h - 1
	e.scrollToCursor(visibleRows, w)

	e.drawRuler(screen, x, y, w)

	style := tcell.StyleDefault.Background(theme.WindowBg).Foreground(theme.WindowFg)
	for row := range visibleRows {
		ly := y + 1 + row
		lineIdx := e.topLine + row
		var line []rune
		if lineIdx < len(e.lines) {
			line = []rune(e.lines[lineIdx])
		}
		for col := range w {
			srcCol := e.hOffset + col
			ch := ' '
			if srcCol < len(line) {
				ch = line[srcCol]
			}
			screen.SetContent(x+col, ly, ch, nil, style)
		}
	}
}

func (e *Editor) drawRuler(screen tcell.Screen, x, y, width int) {
	normal := tcell.StyleDefault.Background(theme.WindowBg).Foreground(theme.WindowFg)
	accent := tcell.StyleDefault.Background(theme.WindowBg).Foreground(theme.WindowAccent)

	labels := make(map[int]rune, width)
	for sx := range width {
		col := e.hOffset + sx + 1
		if col%10 == 0 {
			for i, r := range strconv.Itoa(col / 10) {
				if sx+i < width {
					labels[sx+i] = r
				}
			}
		}
	}

	for sx := range width {
		col := e.hOffset + sx + 1
		switch {
		case sx == 0:
			screen.SetContent(x+sx, y, '▶', nil, accent)
		case labels[sx] != 0:
			screen.SetContent(x+sx, y, labels[sx], nil, normal)
		case col%e.tabWidth == 0:
			screen.SetContent(x+sx, y, '▼', nil, accent)
		default:
			screen.SetContent(x+sx, y, '·', nil, normal)
		}
	}
}

func (e *Editor) scrollToCursor(visibleRows, visibleCols int) {
	if visibleRows > 0 {
		if e.cursorRow < e.topLine {
			e.topLine = e.cursorRow
		}
		if e.cursorRow >= e.topLine+visibleRows {
			e.topLine = e.cursorRow - visibleRows + 1
		}
	}
	if e.topLine < 0 {
		e.topLine = 0
	}
	if visibleCols > 0 {
		if e.cursorCol < e.hOffset {
			e.hOffset = e.cursorCol
		}
		if e.cursorCol >= e.hOffset+visibleCols {
			e.hOffset = e.cursorCol - visibleCols + 1
		}
	}
	if e.hOffset < 0 {
		e.hOffset = 0
	}
}

func (e *Editor) HandleKey(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyRune:
		e.insertRune(event.Rune())
	case tcell.KeyEnter:
		e.insertNewline()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.backspace()
	case tcell.KeyDelete:
		e.deleteForward()
	case tcell.KeyLeft:
		e.moveLeft()
	case tcell.KeyRight:
		e.moveRight()
	case tcell.KeyUp:
		e.moveUp()
	case tcell.KeyDown:
		e.moveDown()
	case tcell.KeyHome:
		e.cursorCol = 0
	case tcell.KeyEnd:
		e.cursorCol = len([]rune(e.lines[e.cursorRow]))
	case tcell.KeyTab:
		next := ((e.cursorCol / e.tabWidth) + 1) * e.tabWidth
		for e.cursorCol < next {
			e.insertRune(' ')
		}
	}
}

func (e *Editor) insertRune(r rune) {
	line := []rune(e.lines[e.cursorRow])
	col := min(e.cursorCol, len(line))
	line = append(line[:col:col], append([]rune{r}, line[col:]...)...)
	e.lines[e.cursorRow] = string(line)
	e.cursorCol = col + 1
}

func (e *Editor) insertNewline() {
	line := []rune(e.lines[e.cursorRow])
	col := min(e.cursorCol, len(line))
	before, after := string(line[:col]), string(line[col:])
	e.lines[e.cursorRow] = before
	tail := append([]string{after}, e.lines[e.cursorRow+1:]...)
	e.lines = append(e.lines[:e.cursorRow+1], tail...)
	e.cursorRow++
	e.cursorCol = 0
}

func (e *Editor) backspace() {
	if e.cursorCol > 0 {
		line := []rune(e.lines[e.cursorRow])
		col := min(e.cursorCol, len(line))
		line = append(line[:col-1], line[col:]...)
		e.lines[e.cursorRow] = string(line)
		e.cursorCol = col - 1
		return
	}
	if e.cursorRow == 0 {
		return
	}
	prevLen := len([]rune(e.lines[e.cursorRow-1]))
	e.lines[e.cursorRow-1] += e.lines[e.cursorRow]
	e.lines = append(e.lines[:e.cursorRow], e.lines[e.cursorRow+1:]...)
	e.cursorRow--
	e.cursorCol = prevLen
}

func (e *Editor) deleteForward() {
	line := []rune(e.lines[e.cursorRow])
	if e.cursorCol < len(line) {
		line = append(line[:e.cursorCol], line[e.cursorCol+1:]...)
		e.lines[e.cursorRow] = string(line)
		return
	}
	if e.cursorRow == len(e.lines)-1 {
		return
	}
	e.lines[e.cursorRow] += e.lines[e.cursorRow+1]
	e.lines = append(e.lines[:e.cursorRow+1], e.lines[e.cursorRow+2:]...)
}

func (e *Editor) moveLeft() {
	if e.cursorCol > 0 {
		e.cursorCol--
		return
	}
	if e.cursorRow > 0 {
		e.cursorRow--
		e.cursorCol = len([]rune(e.lines[e.cursorRow]))
	}
}

func (e *Editor) moveRight() {
	lineLen := len([]rune(e.lines[e.cursorRow]))
	if e.cursorCol < lineLen {
		e.cursorCol++
		return
	}
	if e.cursorRow < len(e.lines)-1 {
		e.cursorRow++
		e.cursorCol = 0
	}
}

func (e *Editor) moveUp() {
	if e.cursorRow == 0 {
		return
	}
	e.cursorRow--
	e.cursorCol = min(e.cursorCol, len([]rune(e.lines[e.cursorRow])))
}

func (e *Editor) moveDown() {
	if e.cursorRow >= len(e.lines)-1 {
		return
	}
	e.cursorRow++
	e.cursorCol = min(e.cursorCol, len([]rune(e.lines[e.cursorRow])))
}

// HandleClick moves the cursor to the clicked position. cy==0 is the
// ruler row and is ignored.
func (e *Editor) HandleClick(cx, cy int) {
	if cy <= 0 {
		return
	}
	row := e.topLine + cy - 1
	row = max(0, min(row, len(e.lines)-1))
	col := max(0, e.hOffset+cx)
	col = min(col, len([]rune(e.lines[row])))
	e.cursorRow, e.cursorCol = row, col
}

func (e *Editor) VScrollMetrics() (offset, visible, total int) {
	visible = max(e.lastH-1, 0)
	total = max(len(e.lines), visible)
	return e.topLine, visible, total
}

func (e *Editor) HScrollMetrics() (offset, visible, total int) {
	visible = max(e.lastW, 0)
	lineLen := len([]rune(e.lines[e.cursorRow]))
	total = max(lineLen+1, visible)
	return e.cursorCol, visible, total
}

func (e *Editor) CursorPos() (cx, cy int, show bool) {
	row := e.cursorRow - e.topLine
	col := e.cursorCol - e.hOffset
	if row < 0 || col < 0 {
		return 0, 0, false
	}
	return col, row + 1, true
}
