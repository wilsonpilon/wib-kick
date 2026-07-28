package tv

import (
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// ParseLabel splits a label containing a single '&' marker (e.g. "S&K") into
// the plain text without the marker and the rune index, within the plain
// text, of the character that should be highlighted as the hotkey. If there
// is no '&', hotkeyIndex is -1. A doubled "&&" is an escape for a literal
// '&' in the plain text, e.g. "&Date && Time formats..." has hotkey 'D' and
// plain text "Date & Time formats...".
func ParseLabel(label string) (plain string, hotkeyIndex int) {
	hotkeyIndex = -1
	runes := []rune(label)
	out := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r != '&' {
			out = append(out, r)
			continue
		}
		if i+1 < len(runes) && runes[i+1] == '&' {
			out = append(out, '&')
			i++
			continue
		}
		if i+1 < len(runes) {
			hotkeyIndex = len(out)
		}
	}
	return string(out), hotkeyIndex
}

// hotkeyRune returns the lowercase rune at hotkeyIndex within plain, or 0 if
// there is no hotkey.
func hotkeyRune(plain string, hotkeyIndex int) rune {
	if hotkeyIndex < 0 {
		return 0
	}
	runes := []rune(plain)
	if hotkeyIndex >= len(runes) {
		return 0
	}
	return unicode.ToLower(runes[hotkeyIndex])
}

// drawLabel writes plain text at (x, y), rendering the rune at hotkeyIndex
// with accent style instead of normal style. Returns the number of cells
// written.
func drawLabel(screen tcell.Screen, x, y int, plain string, hotkeyIndex int, normal, accent tcell.Style) int {
	runes := []rune(plain)
	for i, r := range runes {
		style := normal
		if i == hotkeyIndex {
			style = accent
		}
		screen.SetContent(x+i, y, r, nil, style)
	}
	return len(runes)
}
