package tv

import (
	"strings"

	"github.com/gdamore/tcell/v2"

	"sidekick/internal/theme"
)

// StatusBar renders a single line of hint text at the bottom of the screen.
// Segments wrapped in '~' are drawn with the accent color, e.g. "~Alt+K~
// Menu" draws "Alt+K" in accent and " Menu" in the normal status color.
type StatusBar struct {
	text string
}

func NewStatusBar(text string) *StatusBar {
	return &StatusBar{text: text}
}

func (s *StatusBar) Draw(screen tcell.Screen, x, y, width int) {
	normal := tcell.StyleDefault.Background(theme.StatusBg).Foreground(theme.StatusFg)
	accent := tcell.StyleDefault.Background(theme.StatusBg).Foreground(theme.StatusAccent)

	for i := range width {
		screen.SetContent(x+i, y, ' ', nil, normal)
	}

	col := x + 1
	accented := false
	for part := range strings.SplitSeq(s.text, "~") {
		style := normal
		if accented {
			style = accent
		}
		for _, r := range part {
			screen.SetContent(col, y, r, nil, style)
			col++
		}
		accented = !accented
	}
}
