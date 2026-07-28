// Package theme centraliza as cores do visual estilo Turbo Vision/SideKick.
package theme

import "github.com/gdamore/tcell/v2"

var (
	// Menu bar (topo)
	MenuBg     = tcell.ColorSilver
	MenuFg     = tcell.ColorBlack
	MenuAccent = tcell.ColorRed

	// Desktop (fundo quadriculado)
	DesktopBg = tcell.ColorSilver
	DesktopFg = tcell.ColorBlue
	DesktopCh = '▒'

	// Status bar (rodapé)
	StatusBg     = tcell.ColorSilver
	StatusFg     = tcell.ColorBlack
	StatusAccent = tcell.ColorRed

	// Floating windows (Notepad, etc.)
	WindowBg     = tcell.ColorBlue
	WindowFg     = tcell.ColorYellow
	WindowBorder = tcell.ColorWhite
	WindowAccent = tcell.ColorGreen
	ShadowBg     = tcell.ColorBlack
)
