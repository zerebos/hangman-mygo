package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Window is what the header's own window controls need from the window.
// *mygo.Window has all of it.
type Window interface {
	Minimize()
	Maximize()
	Unmaximize()
	IsMaximized() bool
	Close()
}

// SetWindow gives the view the window to draw controls for, once it exists.
// Without one (on macOS, where the system's traffic lights stay, and in
// tests) the header draws none.
func (a *App) SetWindow(w Window) { a.window = w }

// controlW is how wide each window control is, as Windows draws its own.
const controlW = 46

var (
	iconMinimize = thin(`<path d="M5 12h14"/>`)
	iconMaximize = thin(`<rect x="5" y="5" width="14" height="14" rx="1"/>`)
	iconRestore  = thin(`<rect x="5" y="8" width="11" height="11" rx="1"/><path d="M8 8V6a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-2"/>`)
	iconShut     = thin(`<path d="M6 6l12 12"/><path d="M18 6 6 18"/>`)
)

// thin is an icon in the hairline stroke of the system's window controls.
func thin(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// windowControls draws minimize, maximize and close at the header's full
// height, in place of the system's shorter ones. Maximize is marked as the
// window's maximize button, so Windows 11 shows its snap layouts over it.
func (a *App) windowControls(c *ui.Context, p palette, height float32) {
	w := a.window
	if w == nil {
		return
	}
	control := func(key, label string, icon *ui.SVG, hover, hoverText ui.Color) ui.Element {
		b := ui.ButtonBase(c.Key(key)).Label(label).Width(controlW).Height(height).Center().TextColor(p.Text)
		if b.Hovered() {
			b.Background(hover).TextColor(hoverText)
		}
		b.Transition(ui.ElementTransition{Colors: true, Duration: 90 * time.Millisecond})
		return b.Children(func() { ui.Icon(c, icon).FontSize(16) })
	}
	ui.Row(c).Height(height).Children(func() {
		if control("minimize", "Minimize", iconMinimize, p.SurfaceHover, p.Text).Clicked() {
			w.Minimize()
		}
		maximized := w.IsMaximized()
		label, icon := "Maximize", iconMaximize
		if maximized {
			label, icon = "Restore", iconRestore
		}
		if control("maximize", label, icon, p.SurfaceHover, p.Text).MaximizeButton().Clicked() {
			if maximized {
				w.Unmaximize()
			} else {
				w.Maximize()
			}
		}
		if control("close", "Close", iconShut, ui.Hex("#c42b1c"), ui.Hex("#ffffff")).Clicked() {
			w.Close()
		}
	})
}
