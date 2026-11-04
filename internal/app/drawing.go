package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/gallows"
)

// drawGallows paints the gallows and as much of the victim as wrong guesses
// out of budget have earned. Package gallows is the geometry; this only maps
// it into the element's box and colors it from the theme.
//
// The frame stays quiet in the muted text color, the rope is yellow, and the
// victim is drawn in the text color, the strongest contrast there is,
// because it is what you are meant to watch. When the last guess goes, the
// whole figure turns red.
func drawGallows(c *ui.Context, p palette, wrong, budget int) {
	lost := budget > 0 && wrong >= budget
	figure := p.Text
	if lost {
		figure = p.Danger
	}
	colors := map[gallows.Ink]ui.Color{
		gallows.Frame:  p.TextMuted,
		gallows.Rope:   p.yellow,
		gallows.Figure: figure,
		gallows.Face:   figure,
	}

	box := ui.Box(c.Key("gallows")).FillWidth().Grow(1).MinHeight(160)
	// The newest part draws itself on, replayed on every wrong guess.
	progress := float32(1)
	if wrong > 0 {
		progress = once(box, "draw", wrong, drawOn, ui.EaseInOut)
	}
	box.Draw(func(pt *ui.Painter, r ui.Rect) {
		fit := gallows.Fit(r.W, r.H)
		if fit.Scale <= 0 {
			return
		}
		strokes := append(gallows.FrameStrokes(), gallows.FigureStrokes(budget, wrong, progress)...)
		for _, s := range strokes {
			if len(s.Points) < 2 {
				continue
			}
			var path ui.Path
			for i, pt := range s.Points {
				m := fit.Map(pt)
				if i == 0 {
					path.MoveTo(r.X+m.X, r.Y+m.Y)
				} else {
					path.LineTo(r.X+m.X, r.Y+m.Y)
				}
			}
			if s.Closed {
				path.Close()
			}
			pt.StrokePath(&path, fit.LineWidth(s.Width), colors[s.Ink])
		}
	})
}
