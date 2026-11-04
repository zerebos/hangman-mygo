package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/gallows"
)

const (
	// swayTime is how long the figure swings after a new part.
	swayTime = 2200 * time.Millisecond
	// swayAngle is how far it swings at first, in degrees, and swayLost how
	// far on the guess that loses the word.
	swayAngle = 4.5
	swayLost  = 11
	// swayCycles is how many swings it makes before it comes to rest.
	swayCycles = 3.5
	// penDot is the radius of the pen tip leading a stroke being drawn, in
	// design units.
	penDot = 5
)

// drawGallows paints the gallows and as much of the victim as wrong guesses
// out of budget have earned. Package gallows is the geometry; this only maps
// it into the element's box, colors it from the theme, and moves it.
//
// The frame stays quiet in the muted text color, the rope is yellow, and the
// victim is drawn in the text color, the strongest contrast there is,
// because it is what you are meant to watch. When the last guess goes, the
// whole figure turns red.
//
// Two things move. The newest part draws itself on behind a pen tip, and the
// rope and the figure swing from the beam as the part lands, harder on the
// guess that loses the word, settling as a pendulum does.
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

	box := ui.Box(c.Key("gallows")).FillWidth().Grow(1).MinHeight(160).Transition(glide)
	progress, swing := float32(1), float32(0)
	if wrong > 0 {
		progress = once(box, "draw", wrong, drawOn, ui.EaseOut)
		t := once(box, "sway", wrong, swayTime, ui.Linear)
		angle := float32(swayAngle)
		if lost {
			angle = swayLost
		}
		// A sine whose swing dies away, so it starts at rest, swings out as
		// the part lands, and ends exactly where it started.
		swing = angle * (1 - t) * (1 - t) * float32(math.Sin(float64(t)*swayCycles*2*math.Pi))
	}
	settled := gallows.PartsDrawn(budget, max(wrong-1, 0))

	box.Draw(func(pt *ui.Painter, r ui.Rect) {
		fit := gallows.Fit(r.W, r.H)
		if fit.Scale <= 0 {
			return
		}
		at := func(q gallows.Point) (float32, float32) {
			m := fit.Map(q)
			return r.X + m.X, r.Y + m.Y
		}
		stroke := func(s gallows.Stroke) {
			if len(s.Points) < 2 {
				return
			}
			var path ui.Path
			for i, q := range s.Points {
				x, y := at(q)
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			if s.Closed {
				path.Close()
			}
			pt.StrokePath(&path, fit.LineWidth(s.Width), colors[s.Ink])
		}

		var hanging []gallows.Stroke
		for _, s := range gallows.FrameStrokes() {
			if s.Ink == gallows.Rope {
				hanging = append(hanging, s)
			} else {
				stroke(s)
			}
		}
		figureStrokes := gallows.FigureStrokes(budget, wrong, progress)
		hanging = append(hanging, figureStrokes...)

		px, py := at(gallows.Pivot())
		pt.Transform(ui.Rotation(swing), px, py, func() {
			for _, s := range hanging {
				stroke(s)
			}
			// The pen tip rides the end of each stroke still being drawn.
			if progress < 1 {
				for _, s := range figureStrokes[min(settled, len(figureStrokes)):] {
					if len(s.Points) == 0 {
						continue
					}
					x, y := at(s.Points[len(s.Points)-1])
					var dot ui.Path
					dot.Circle(x, y, penDot*fit.Scale*(1-progress*0.5))
					pt.FillPath(&dot, colors[s.Ink])
				}
			}
		})
	})
}
