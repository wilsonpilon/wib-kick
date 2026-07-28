package tv

import (
	"math"
	"strconv"

	"github.com/gdamore/tcell/v2"

	"sidekick/internal/theme"
)

const (
	closeBoxWidth = 3 // "[■]"
	zoomBoxWidth  = 3 // "[↑]" or "[↕]"

	minWindowW = 14
	minWindowH = 5
)

// WindowContent is whatever a floating Window shows in its interior. The
// generic window chrome (border, shadow, title bar, scrollbars) knows
// nothing about what kind of accessory it's hosting.
type WindowContent interface {
	Draw(screen tcell.Screen, x, y, w, h int)
	HandleKey(event *tcell.EventKey)
	HandleClick(cx, cy int) // cx,cy relative to the content rect
	VScrollMetrics() (offset, visible, total int)
	HScrollMetrics() (offset, visible, total int)
	CursorPos() (cx, cy int, show bool) // relative to the content rect
}

type dragMode int

const (
	dragNone dragMode = iota
	dragMove
	dragResize
)

// Window is a floating, Turbo Vision-style window: double-line border,
// drop shadow, title bar with close/zoom buttons, and scrollbars.
type Window struct {
	id      int
	title   string
	content WindowContent

	x, y, w, h                 int
	prevX, prevY, prevW, prevH int
	zoomed                     bool

	dragKind           dragMode
	dragOffX, dragOffY int
}

func NewWindow(id int, title string, x, y, w, h int, content WindowContent) *Window {
	return &Window{id: id, title: title, content: content, x: x, y: y, w: w, h: h}
}

func (win *Window) closeBoxX() int    { return win.x + 1 }
func (win *Window) zoomBoxX() int     { return win.x + win.w - 1 - zoomBoxWidth }
func (win *Window) numberStr() string { return strconv.Itoa(win.id) }
func (win *Window) numberX() int {
	return win.zoomBoxX() - 1 - len(win.numberStr())
}

// Draw renders the border, title bar, scrollbars and content.
func (win *Window) Draw(screen tcell.Screen) {
	borderStyle := tcell.StyleDefault.Background(theme.WindowBg).Foreground(theme.WindowBorder)
	accentStyle := tcell.StyleDefault.Background(theme.WindowBg).Foreground(theme.WindowAccent)

	drawDoubleBorder(screen, win.x, win.y, win.w, win.h, borderStyle)

	// Close box: [■]
	cx := win.closeBoxX()
	screen.SetContent(cx, win.y, '[', nil, borderStyle)
	screen.SetContent(cx+1, win.y, '■', nil, accentStyle)
	screen.SetContent(cx+2, win.y, ']', nil, borderStyle)

	// Window number + zoom box: N[↑] or N[↕]
	zx := win.zoomBoxX()
	numStr := win.numberStr()
	numX := win.numberX()
	if numX > cx+closeBoxWidth {
		for i, r := range numStr {
			screen.SetContent(numX+i, win.y, r, nil, borderStyle)
		}
	}
	zoomGlyph := '↑'
	if win.zoomed {
		zoomGlyph = '↕'
	}
	screen.SetContent(zx, win.y, '[', nil, borderStyle)
	screen.SetContent(zx+1, win.y, zoomGlyph, nil, accentStyle)
	screen.SetContent(zx+2, win.y, ']', nil, borderStyle)

	// Title text, centered in whatever space remains.
	titleStart := cx + closeBoxWidth + 1
	titleEnd := numX - 1
	if titleEnd > titleStart {
		title := []rune(win.title)
		avail := titleEnd - titleStart
		if len(title) > avail {
			title = title[:avail]
		}
		for i, r := range title {
			screen.SetContent(titleStart+i, win.y, r, nil, borderStyle)
		}
	}

	// Content interior.
	win.content.Draw(screen, win.x+1, win.y+1, win.w-2, win.h-2)

	// Vertical scrollbar (right border), drawn after content.Draw so the
	// content's freshly reported metrics reflect this frame's geometry.
	if trackLen := win.h - 2; trackLen > 0 {
		off, vis, tot := win.content.VScrollMetrics()
		pos, size := scrollbarThumb(trackLen, tot, vis, off)
		for row := range trackLen {
			ch := '║'
			if row >= pos && row < pos+size {
				ch = '█'
			}
			screen.SetContent(win.x+win.w-1, win.y+1+row, ch, nil, borderStyle)
		}
	}

	// Horizontal scrollbar (bottom border).
	if trackLen := win.w - 2; trackLen > 0 {
		off, vis, tot := win.content.HScrollMetrics()
		pos, size := scrollbarThumb(trackLen, tot, vis, off)
		for col := range trackLen {
			ch := '═'
			if col >= pos && col < pos+size {
				ch = '█'
			}
			screen.SetContent(win.x+1+col, win.y+win.h-1, ch, nil, borderStyle)
		}
	}
}

// DrawShadow paints the classic Borland-style drop shadow: a 1-wide strip
// down the right side and a 2-tall strip along the bottom.
func (win *Window) DrawShadow(screen tcell.Screen) {
	style := tcell.StyleDefault.Background(theme.ShadowBg)
	for row := win.y + 2; row <= win.y+win.h+1; row++ {
		screen.SetContent(win.x+win.w, row, ' ', nil, style)
	}
	for row := win.y + win.h; row <= win.y+win.h+1; row++ {
		for col := win.x + 1; col <= win.x+win.w; col++ {
			screen.SetContent(col, row, ' ', nil, style)
		}
	}
}

func (win *Window) Contains(x, y int) bool {
	return x >= win.x && x < win.x+win.w && y >= win.y && y < win.y+win.h
}

func (win *Window) HitClose(x, y int) bool {
	return y == win.y && x >= win.closeBoxX() && x < win.closeBoxX()+closeBoxWidth
}

func (win *Window) HitZoom(x, y int) bool {
	return y == win.y && x >= win.zoomBoxX() && x < win.zoomBoxX()+zoomBoxWidth
}

func (win *Window) HitResizeHandle(x, y int) bool {
	return x == win.x+win.w-1 && y == win.y+win.h-1
}

// HitTitleBar reports whether (x,y) is on the top border row, excluding
// the corners and the close/zoom buttons.
func (win *Window) HitTitleBar(x, y int) bool {
	if y != win.y || x <= win.x || x >= win.x+win.w-1 {
		return false
	}
	return !win.HitClose(x, y) && !win.HitZoom(x, y)
}

// HitContent reports whether (x,y) is inside the content rect, returning
// coordinates relative to that rect.
func (win *Window) HitContent(x, y int) (cx, cy int, ok bool) {
	ix, iy, iw, ih := win.x+1, win.y+1, win.w-2, win.h-2
	if x < ix || x >= ix+iw || y < iy || y >= iy+ih {
		return 0, 0, false
	}
	return x - ix, y - iy, true
}

// CursorScreenPos translates the content's own CursorPos() into absolute
// screen coordinates, for App to place the terminal cursor.
func (win *Window) CursorScreenPos() (x, y int, ok bool) {
	cx, cy, show := win.content.CursorPos()
	if !show {
		return 0, 0, false
	}
	return win.x + 1 + cx, win.y + 1 + cy, true
}

func (win *Window) BeginMove(mouseX, mouseY int) {
	if win.zoomed {
		win.zoomed = false
		win.w, win.h = win.prevW, win.prevH
	}
	win.dragKind = dragMove
	win.dragOffX = mouseX - win.x
	win.dragOffY = mouseY - win.y
}

func (win *Window) BeginResize() {
	win.zoomed = false
	win.dragKind = dragResize
}

func (win *Window) DragTo(mouseX, mouseY, deskX, deskY, deskW, deskH int) {
	switch win.dragKind {
	case dragMove:
		nx := max(deskX, min(mouseX-win.dragOffX, deskX+deskW-win.w))
		ny := max(deskY, min(mouseY-win.dragOffY, deskY+deskH-win.h))
		win.x, win.y = nx, ny
	case dragResize:
		nw := max(minWindowW, min(mouseX-win.x+1, deskX+deskW-win.x-1))
		nh := max(minWindowH, min(mouseY-win.y+1, deskY+deskH-win.y-2))
		win.w, win.h = nw, nh
	}
}

func (win *Window) EndDrag() {
	win.dragKind = dragNone
}

func (win *Window) ToggleZoom(deskX, deskY, deskW, deskH int) {
	if win.zoomed {
		win.x, win.y, win.w, win.h = win.prevX, win.prevY, win.prevW, win.prevH
		win.zoomed = false
		return
	}
	win.prevX, win.prevY, win.prevW, win.prevH = win.x, win.y, win.w, win.h
	win.x, win.y = deskX, deskY
	win.w, win.h = deskW-1, deskH-2
	win.zoomed = true
}

// drawDoubleBorder draws a classic MS-DOS double-line box (╔═╗║╚╝).
func drawDoubleBorder(screen tcell.Screen, x, y, w, h int, style tcell.Style) {
	screen.SetContent(x, y, '╔', nil, style)
	screen.SetContent(x+w-1, y, '╗', nil, style)
	screen.SetContent(x, y+h-1, '╚', nil, style)
	screen.SetContent(x+w-1, y+h-1, '╝', nil, style)
	for col := 1; col < w-1; col++ {
		screen.SetContent(x+col, y, '═', nil, style)
		screen.SetContent(x+col, y+h-1, '═', nil, style)
	}
	for row := 1; row < h-1; row++ {
		screen.SetContent(x, y+row, '║', nil, style)
		screen.SetContent(x+w-1, y+row, '║', nil, style)
	}
}

// scrollbarThumb computes a proportional scrollbar thumb's position and
// size along a track of trackLen cells.
func scrollbarThumb(trackLen, total, visible, offset int) (pos, size int) {
	if trackLen <= 0 {
		return 0, 0
	}
	if total < visible {
		total = visible
	}
	if total <= 0 {
		total = 1
	}
	size = int(math.Round(float64(visible) / float64(total) * float64(trackLen)))
	size = max(1, min(size, trackLen))

	maxOffset := max(total-visible, 0)
	if maxOffset == 0 {
		return 0, size
	}
	pos = int(math.Round(float64(offset) / float64(maxOffset) * float64(trackLen-size)))
	pos = max(0, min(pos, trackLen-size))
	return pos, size
}
