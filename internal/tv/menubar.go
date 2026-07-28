package tv

import (
	"unicode"

	"github.com/gdamore/tcell/v2"

	"sidekick/internal/theme"
)

// menuLevel is one open dropdown or flyout: a list of items plus which one
// is currently highlighted, and the rect it was last drawn at (for mouse
// hit-testing).
type menuLevel struct {
	items    []*item
	selected int

	x, y, w, h int
}

// menuTitle is a single top-level Turbo Vision style menu title (e.g. "S&K")
// together with its own dropdown items and currently open flyout stack.
type menuTitle struct {
	titlePlain  string
	titleHotIdx int
	items       []*item

	stack []*menuLevel

	x, y, w int // last-drawn title rect, for hit-testing
}

// title creates a top-level menu title and its dropdown items. Labels use
// '&' to mark the hotkey letter ("&&" for a literal '&').
func title(label string, items ...*item) *menuTitle {
	plain, idx := ParseLabel(label)
	return &menuTitle{titlePlain: plain, titleHotIdx: idx, items: items}
}

func (t *menuTitle) hotkeyRune() rune { return hotkeyRune(t.titlePlain, t.titleHotIdx) }

func (t *menuTitle) open()  { t.stack = []*menuLevel{{items: t.items, selected: firstSelectable(t.items)}} }
func (t *menuTitle) close() { t.stack = nil }

// MenuBar is a row of top-level Turbo Vision style menu titles. Each title
// opens a dropdown via Alt+hotkey, mouse click, or keyboard navigation, and
// dropdown items may themselves be submenus, which open cascading flyouts to
// the side. At most one title is open at a time.
type MenuBar struct {
	titles []*menuTitle
	open   int // index into titles of the open one, or -1 if closed
}

// NewMenuBar creates a menu bar from a row of titles, in left-to-right
// order. See title() for constructing each one.
func NewMenuBar(titles ...*menuTitle) *MenuBar {
	return &MenuBar{titles: titles, open: -1}
}

func (m *MenuBar) IsOpen() bool { return m.open >= 0 }

// Close hides the menu entirely, including any open flyouts.
func (m *MenuBar) Close() {
	if m.open >= 0 {
		m.titles[m.open].close()
		m.open = -1
	}
}

// OpenByHotkey opens the title whose Alt-accelerator matches r, switching
// directly to it even if a different title is already open. Reports whether
// a title matched.
func (m *MenuBar) OpenByHotkey(r rune) bool {
	r = unicode.ToLower(r)
	for i, t := range m.titles {
		if t.hotkeyRune() == r {
			if m.open >= 0 && m.open != i {
				m.titles[m.open].close()
			}
			m.open = i
			t.open()
			return true
		}
	}
	return false
}

// ToggleTitle opens the title at index, or closes the menu if that title is
// already open. Used for mouse clicks on the title bar.
func (m *MenuBar) ToggleTitle(index int) {
	if index < 0 || index >= len(m.titles) {
		return
	}
	wasOpen := m.open == index
	m.Close()
	if !wasOpen {
		m.open = index
		m.titles[index].open()
	}
}

// switchTitle closes the currently open title and opens the one dir (+1/-1)
// steps away, wrapping around.
func (m *MenuBar) switchTitle(dir int) {
	n := len(m.titles)
	if n == 0 || m.open < 0 {
		return
	}
	m.titles[m.open].close()
	m.open = ((m.open+dir)%n + n) % n
	m.titles[m.open].open()
}

func firstSelectable(items []*item) int {
	for i, it := range items {
		if it.selectable() {
			return i
		}
	}
	return -1
}

// nextSelectable returns the next selectable index from "from" moving by
// dir (+1/-1), wrapping around and skipping separators.
func nextSelectable(items []*item, from, dir int) int {
	n := len(items)
	if n == 0 {
		return -1
	}
	i := from
	for range n {
		i = ((i+dir)%n + n) % n
		if items[i].selectable() {
			return i
		}
	}
	return from
}

// DrawBar renders the menu title strip across [x, x+width) at row y. Call
// DrawOverlay afterwards, once everything below the bar (e.g. the desktop)
// has been drawn, so an open dropdown isn't painted over.
func (m *MenuBar) DrawBar(screen tcell.Screen, x, y, width int) {
	normal := tcell.StyleDefault.Background(theme.MenuBg).Foreground(theme.MenuFg)
	accent := tcell.StyleDefault.Background(theme.MenuBg).Foreground(theme.MenuAccent)

	for i := range width {
		screen.SetContent(x+i, y, ' ', nil, normal)
	}

	cx := x
	for _, t := range m.titles {
		t.x, t.y = cx, y
		t.w = len([]rune(t.titlePlain)) + 2
		drawLabel(screen, cx+1, y, t.titlePlain, t.titleHotIdx, normal, accent)
		cx += t.w
	}
}

// DrawOverlay renders the open title's dropdown/flyout levels, clipped to
// fit within [x, x+width) x [y, y+height). Must be called after the
// primitives below the menu bar are drawn, so the dropdown appears on top of
// them.
func (m *MenuBar) DrawOverlay(screen tcell.Screen, x, y, width, height int) {
	if m.open < 0 {
		return
	}
	t := m.titles[m.open]
	for li, lvl := range t.stack {
		anchorX, anchorY := anchorFor(t, li, y)
		contentW, contentH := computeLevelSize(lvl.items)
		w, h := contentW+2, contentH+2

		drawX := anchorX
		if drawX+w > x+width {
			if li == 0 {
				drawX = x + width - w
			} else {
				drawX = t.stack[li-1].x - w
			}
			if drawX < x {
				drawX = x
			}
		}
		drawY := anchorY
		if drawY+h > y+height {
			drawY = y + height - h
		}
		if drawY < y+1 {
			drawY = y + 1
		}

		lvl.x, lvl.y, lvl.w, lvl.h = drawX, drawY, w, h
		drawLevel(screen, lvl)
	}
}

// anchorFor returns the preferred (unclipped) top-left corner for level li
// of title t: level 0 opens below the title, deeper levels open to the
// right of the row that was expanded to reach them.
func anchorFor(t *menuTitle, li, barY int) (x, y int) {
	if li == 0 {
		return t.x, barY + 1
	}
	parent := t.stack[li-1]
	return parent.x + parent.w, parent.y + 1 + parent.selected
}

func computeLevelSize(items []*item) (width, height int) {
	maxLen := 0
	hasSubmenu := false
	for _, it := range items {
		if it.kind == kindSeparator {
			continue
		}
		need := len([]rune(it.plain))
		if it.shortcut != "" {
			need += 2 + len([]rune(it.shortcut))
		}
		if need > maxLen {
			maxLen = need
		}
		if it.kind == kindSubmenu {
			hasSubmenu = true
		}
	}
	width = maxLen + 2
	if hasSubmenu {
		width += 2
	}
	return width, len(items)
}

// drawBorder draws a single-line box (classic MS-DOS style) around
// [x, x+w) x [y, y+h).
func drawBorder(screen tcell.Screen, x, y, w, h int, style tcell.Style) {
	screen.SetContent(x, y, '┌', nil, style)
	screen.SetContent(x+w-1, y, '┐', nil, style)
	screen.SetContent(x, y+h-1, '└', nil, style)
	screen.SetContent(x+w-1, y+h-1, '┘', nil, style)
	for col := 1; col < w-1; col++ {
		screen.SetContent(x+col, y, '─', nil, style)
		screen.SetContent(x+col, y+h-1, '─', nil, style)
	}
	for row := 1; row < h-1; row++ {
		screen.SetContent(x, y+row, '│', nil, style)
		screen.SetContent(x+w-1, y+row, '│', nil, style)
	}
}

func drawLevel(screen tcell.Screen, lvl *menuLevel) {
	normal := tcell.StyleDefault.Background(theme.MenuBg).Foreground(theme.MenuFg)
	accent := tcell.StyleDefault.Background(theme.MenuBg).Foreground(theme.MenuAccent)
	selNormal := tcell.StyleDefault.Background(theme.MenuFg).Foreground(theme.MenuBg)
	selAccent := tcell.StyleDefault.Background(theme.MenuFg).Foreground(theme.MenuAccent)

	drawBorder(screen, lvl.x, lvl.y, lvl.w, lvl.h, normal)

	hasSubmenu := false
	for _, it := range lvl.items {
		if it.kind == kindSubmenu {
			hasSubmenu = true
			break
		}
	}

	contentX, contentW := lvl.x+1, lvl.w-2
	for row, it := range lvl.items {
		y := lvl.y + 1 + row
		n, a := normal, accent
		if row == lvl.selected {
			n, a = selNormal, selAccent
		}

		for col := range contentW {
			screen.SetContent(contentX+col, y, ' ', nil, n)
		}

		switch it.kind {
		case kindSeparator:
			for col := range contentW {
				screen.SetContent(contentX+col, y, '─', nil, normal)
			}
		case kindSubmenu:
			drawLabel(screen, contentX+1, y, it.plain, it.hotkeyIndex, n, a)
			if hasSubmenu {
				screen.SetContent(contentX+contentW-2, y, '►', nil, n)
			}
		case kindAction:
			drawLabel(screen, contentX+1, y, it.plain, it.hotkeyIndex, n, a)
			if it.shortcut != "" {
				rightEdge := contentX + contentW - 1
				if hasSubmenu {
					rightEdge -= 2
				}
				sr := []rune(it.shortcut)
				sx := rightEdge - len(sr) + 1
				for i, r := range sr {
					screen.SetContent(sx+i, y, r, nil, n)
				}
			}
		}
	}
}

// HitTitle reports which title (x, y) falls on, as drawn by the last
// DrawBar call.
func (m *MenuBar) HitTitle(x, y int) (index int, ok bool) {
	for i, t := range m.titles {
		if y == t.y && x >= t.x && x < t.x+t.w {
			return i, true
		}
	}
	return -1, false
}

// HitItem reports which open level and item index (x, y) falls on, checking
// the deepest (topmost) level first since flyouts can overlap their parent.
func (m *MenuBar) HitItem(x, y int) (level, index int, ok bool) {
	if m.open < 0 {
		return -1, -1, false
	}
	stack := m.titles[m.open].stack
	for li := len(stack) - 1; li >= 0; li-- {
		lvl := stack[li]
		if x < lvl.x+1 || x >= lvl.x+lvl.w-1 || y < lvl.y+1 || y >= lvl.y+lvl.h-1 {
			continue
		}
		return li, y - lvl.y - 1, true
	}
	return -1, -1, false
}

// Click selects itemIndex within level, closing any deeper flyouts first,
// and activates it as if Enter had been pressed.
func (m *MenuBar) Click(level, itemIndex int) (activatedID string, activated bool) {
	if m.open < 0 {
		return "", false
	}
	t := m.titles[m.open]
	if level < 0 || level >= len(t.stack) {
		return "", false
	}
	t.stack = t.stack[:level+1]
	lvl := t.stack[level]
	if itemIndex < 0 || itemIndex >= len(lvl.items) || !lvl.items[itemIndex].selectable() {
		return "", false
	}
	lvl.selected = itemIndex
	return m.activateSelected(t, lvl)
}

// HandleKey processes a key event while the menu is open. It returns the
// activated item's id, if any.
func (m *MenuBar) HandleKey(event *tcell.EventKey) (activatedID string, activated bool) {
	if m.open < 0 {
		return "", false
	}
	t := m.titles[m.open]
	top := t.stack[len(t.stack)-1]

	switch event.Key() {
	case tcell.KeyEscape:
		if len(t.stack) > 1 {
			t.stack = t.stack[:len(t.stack)-1]
		} else {
			m.Close()
		}
	case tcell.KeyUp:
		top.selected = nextSelectable(top.items, top.selected, -1)
	case tcell.KeyDown:
		top.selected = nextSelectable(top.items, top.selected, 1)
	case tcell.KeyRight:
		if len(t.stack) == 1 && len(m.titles) > 1 {
			m.switchTitle(1)
		} else {
			m.openSubmenu(t, top)
		}
	case tcell.KeyLeft:
		if len(t.stack) > 1 {
			t.stack = t.stack[:len(t.stack)-1]
		} else if len(m.titles) > 1 {
			m.switchTitle(-1)
		}
	case tcell.KeyEnter:
		return m.activateSelected(t, top)
	case tcell.KeyRune:
		r := unicode.ToLower(event.Rune())
		for i, it := range top.items {
			if it.selectable() && it.hotkey() == r {
				top.selected = i
				return m.activateSelected(t, top)
			}
		}
	}
	return "", false
}

func (m *MenuBar) openSubmenu(t *menuTitle, lvl *menuLevel) {
	if lvl.selected < 0 {
		return
	}
	it := lvl.items[lvl.selected]
	if it.kind != kindSubmenu {
		return
	}
	t.stack = append(t.stack, &menuLevel{items: it.children, selected: firstSelectable(it.children)})
}

func (m *MenuBar) activateSelected(t *menuTitle, lvl *menuLevel) (activatedID string, activated bool) {
	if lvl.selected < 0 {
		return "", false
	}
	it := lvl.items[lvl.selected]
	switch it.kind {
	case kindSubmenu:
		m.openSubmenu(t, lvl)
		return "", false
	case kindAction:
		id := it.id
		m.Close()
		return id, true
	}
	return "", false
}
