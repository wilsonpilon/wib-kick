package tv

// itemKind distinguishes the three kinds of entries a menu level can have.
type itemKind int

const (
	kindAction itemKind = iota
	kindSeparator
	kindSubmenu
)

// item is one entry of a menu level: an activatable action, a visual
// separator, or a submenu that opens a cascading flyout.
type item struct {
	kind        itemKind
	plain       string
	hotkeyIndex int
	id          string
	shortcut    string
	children    []*item
}

// action creates a leaf menu entry that activates the command identified by
// id when chosen. An optional shortcut hint (e.g. "Alt+N", "F3") is shown
// right-aligned on the item's row.
func action(label, id string, shortcut ...string) *item {
	plain, idx := ParseLabel(label)
	it := &item{kind: kindAction, plain: plain, hotkeyIndex: idx, id: id}
	if len(shortcut) > 0 {
		it.shortcut = shortcut[0]
	}
	return it
}

// submenu creates a menu entry that opens a cascading flyout of children.
func submenu(label string, children ...*item) *item {
	plain, idx := ParseLabel(label)
	return &item{kind: kindSubmenu, plain: plain, hotkeyIndex: idx, children: children}
}

// separator creates a non-selectable dividing line.
func separator() *item {
	return &item{kind: kindSeparator}
}

func (it *item) selectable() bool {
	return it.kind != kindSeparator
}

func (it *item) hotkey() rune {
	return hotkeyRune(it.plain, it.hotkeyIndex)
}
