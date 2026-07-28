package tv

import (
	"github.com/gdamore/tcell/v2"

	"sidekick/internal/theme"
)

// Desktop fills a rectangular area with the checkered SideKick background.
type Desktop struct{}

func NewDesktop() *Desktop { return &Desktop{} }

func (d *Desktop) Draw(screen tcell.Screen, x, y, width, height int) {
	style := tcell.StyleDefault.Background(theme.DesktopBg).Foreground(theme.DesktopFg)
	for row := range height {
		for col := range width {
			screen.SetContent(x+col, y+row, theme.DesktopCh, nil, style)
		}
	}
}
