package tv

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App is the root primitive of the SideKick clone: it owns the menu bar,
// desktop, status bar and any floating windows, and routes keyboard/mouse
// input between them.
type App struct {
	*tview.Box

	menu    *MenuBar
	desktop *Desktop
	status  *StatusBar

	onExit func()

	windows      []*Window // back-to-front; last = topmost/focused
	notepadCount int
	dragTarget   *Window
	dragMode     dragMode
}

// NewApp creates the root screen. onExit is called when the user activates
// the "Exit" menu item.
func NewApp(onExit func()) *App {
	menu := NewMenuBar(
		title("S&K",
			action("&Notepad", "notepad"),
			separator(),
			action("&Time Planner", "time-planner"),
			action("&Address Book", "address-book"),
			action("&Communications", "communications"),
			separator(),
			action("Ca&lculator", "calculator"),
			separator(),
			submenu("&Services",
				action("&About Sidekick...", "about"),
				separator(),
				action("&Date && Time formats...", "date-time-formats"),
				action("&Mouse...", "mouse"),
				action("S&ystem colors...", "system-colors"),
				separator(),
				action("Screen &Cut", "screen-cut"),
				separator(),
				action("&Save Desktop...", "save-desktop"),
				action("&Restore Desktop...", "restore-desktop"),
				separator(),
				action("&Unload Sidekick...", "unload"),
			),
			action("E&xit", "exit"),
		),
	)

	return &App{
		Box:     tview.NewBox(),
		menu:    menu,
		desktop: NewDesktop(),
		status:  NewStatusBar("~Alt+K~ Menu"),
		onExit:  onExit,
	}
}

// fileMenuTitle returns the Time Planner's "&File" menu, parked here until
// per-accessory dynamic menu switching exists. Not currently used.
func fileMenuTitle() *menuTitle {
	return title("&File",
		action("&New...", "new", "Alt+N"),
		action("&Open...", "open", "F3"),
		action("&Defaults...", "defaults"),
		action("Setup reconc&ile...", "setup-reconcile"),
		action("Reconcile day", "reconcile-day", "Alt+R"),
		submenu("&Tools",
			action("&Copy...", "copy"),
			action("&Rename...", "rename"),
			action("&Delete", "delete"),
		),
		separator(),
		action("Prot&ect", "protect"),
		action("Unprotect", "unprotect"),
		separator(),
		action("&Print...", "print", "F4"),
		action("Printer settin&gs...", "printer-settings", "Alt+G"),
		separator(),
		action("&Close", "close"),
	)
}

// desktopRect returns the rect occupied by the desktop, between the menu
// bar and the status bar.
func (a *App) desktopRect() (x, y, w, h int) {
	rx, ry, width, height := a.GetRect()
	return rx, ry + 1, width, height - 2
}

func (a *App) focusedWindow() *Window {
	if len(a.windows) == 0 {
		return nil
	}
	return a.windows[len(a.windows)-1]
}

func (a *App) raiseWindow(win *Window) {
	for i, w := range a.windows {
		if w == win {
			a.windows = append(a.windows[:i], a.windows[i+1:]...)
			a.windows = append(a.windows, win)
			return
		}
	}
}

func (a *App) closeWindow(win *Window) {
	for i, w := range a.windows {
		if w == win {
			a.windows = append(a.windows[:i], a.windows[i+1:]...)
			return
		}
	}
	if a.dragTarget == win {
		a.dragTarget, a.dragMode = nil, dragNone
	}
}

// cascadeOrigin computes the top-left corner for the n-th (0-based)
// cascaded window of size winW x winH within the desktop rect, wrapping
// back to the base origin once the offset would push the window or its
// shadow outside the desktop.
func cascadeOrigin(n, deskX, deskY, deskW, deskH, winW, winH int) (x, y int) {
	baseX, baseY := deskX+1, deskY
	maxStepsX := max((deskW-1-(winW+1))/2, 0)
	maxStepsY := max((deskH-(winH+2))/1, 0)
	maxSteps := min(maxStepsX, maxStepsY)
	step := n % (maxSteps + 1)
	return baseX + step*2, baseY + step
}

// Draw renders the menu bar, desktop, status bar and any floating windows
// top to bottom, then places the terminal cursor for the focused window.
func (a *App) Draw(screen tcell.Screen) {
	x, y, width, height := a.GetRect()
	if width <= 0 || height <= 0 {
		return
	}

	a.menu.DrawBar(screen, x, y, width)
	if height > 2 {
		a.desktop.Draw(screen, x, y+1, width, height-2)
	}
	if height > 1 {
		a.status.Draw(screen, x, y+height-1, width)
	}

	for _, win := range a.windows {
		win.DrawShadow(screen)
		win.Draw(screen)
	}

	a.menu.DrawOverlay(screen, x, y, width, height)

	if !a.menu.IsOpen() {
		if win := a.focusedWindow(); win != nil {
			if cx, cy, ok := win.CursorScreenPos(); ok {
				screen.ShowCursor(cx, cy)
				return
			}
		}
	}
	screen.HideCursor()
}

// InputHandler opens the S&K menu on Alt+K, and while the menu is open,
// forwards navigation keys to it. Otherwise keys go to the focused window.
func (a *App) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return a.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModAlt != 0 {
			if a.menu.OpenByHotkey(event.Rune()) {
				return
			}
		}

		if a.menu.IsOpen() {
			id, activated := a.menu.HandleKey(event)
			if activated {
				a.activate(id)
			}
			return
		}

		if win := a.focusedWindow(); win != nil {
			win.content.HandleKey(event)
		}
	})
}

// MouseHandler opens/closes the menu and activates its items on click, and
// otherwise routes clicks/drags to floating windows.
func (a *App) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
	return a.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		x, y := event.Position()

		switch action {
		case tview.MouseLeftClick:
			if idx, ok := a.menu.HitTitle(x, y); ok {
				a.menu.ToggleTitle(idx)
				return true, nil
			}

			if a.menu.IsOpen() {
				if level, idx, ok := a.menu.HitItem(x, y); ok {
					id, activated := a.menu.Click(level, idx)
					if activated {
						a.activate(id)
					}
				} else {
					a.menu.Close()
				}
				return true, nil
			}

		case tview.MouseLeftDown:
			if a.menu.IsOpen() {
				return true, nil
			}
			for i := len(a.windows) - 1; i >= 0; i-- {
				win := a.windows[i]
				if !win.Contains(x, y) {
					continue
				}
				a.raiseWindow(win)
				switch {
				case win.HitClose(x, y):
					a.closeWindow(win)
				case win.HitZoom(x, y):
					dx, dy, dw, dh := a.desktopRect()
					win.ToggleZoom(dx, dy, dw, dh)
				case win.HitResizeHandle(x, y):
					a.dragTarget, a.dragMode = win, dragResize
					win.BeginResize()
				case win.HitTitleBar(x, y):
					a.dragTarget, a.dragMode = win, dragMove
					win.BeginMove(x, y)
				default:
					if cx, cy, ok := win.HitContent(x, y); ok {
						win.content.HandleClick(cx, cy)
					}
				}
				return true, nil
			}

		case tview.MouseMove:
			if a.dragTarget != nil {
				dx, dy, dw, dh := a.desktopRect()
				a.dragTarget.DragTo(x, y, dx, dy, dw, dh)
				return true, nil
			}

		case tview.MouseLeftUp:
			if a.dragTarget != nil {
				a.dragTarget.EndDrag()
				a.dragTarget, a.dragMode = nil, dragNone
				return true, nil
			}
		}

		return false, nil
	})
}

func (a *App) activate(id string) {
	switch id {
	case "exit":
		if a.onExit != nil {
			a.onExit()
		}
	case "notepad":
		a.notepadCount++
		dx, dy, dw, dh := a.desktopRect()
		winW, winH := dw-3, dh/2
		wx, wy := cascadeOrigin(a.notepadCount-1, dx, dy, dw, dh, winW, winH)
		win := NewWindow(a.notepadCount, fmt.Sprintf("Untitled %d", a.notepadCount), wx, wy, winW, winH, NewEditor())
		a.windows = append(a.windows, win)
	default:
		// Not yet implemented.
	}
}
