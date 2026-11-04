package app

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/gallows"
	"github.com/zerebos/hangman-mygo/internal/game"
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
	// partStep is how much longer the draw-on takes for each extra part,
	// when giving up spends several at once.
	partStep = 220 * time.Millisecond
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
//
// Every line is roughened as a marker draws it: wobbling, tapered and run
// past its ends.
func (a *App) drawGallows(c *ui.Context, p palette) {
	wrong, budget := a.game.WrongGuesses(), a.game.GuessBudget()
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

	box := a.glideOn(ui.Box(c.Key("gallows")).FillWidth().Grow(1).MinHeight(160)).Label(gallowsLabel(wrong, budget))
	// The parts this frame draws on: the newest one, or every part giving up
	// spent, one after another.
	from := max(wrong-1, 0)
	if a.gaveUpFrom >= 0 && a.game.Result() == game.Lost {
		from = a.gaveUpFrom
	}
	settled := gallows.PartsDrawn(budget, from)
	drawn := gallows.PartsDrawn(budget, wrong)
	progress, swing := float32(1), float32(0)
	if wrong > 0 {
		progress = once(box, "draw", a.run(wrong), drawOn+partStep*time.Duration(max(drawn-settled-1, 0)), ui.EaseOut)
		t := once(box, "sway", a.run(wrong), swayTime, ui.Linear)
		angle := float32(swayAngle)
		if lost {
			angle = swayLost
		}
		// A sine whose swing dies away, so it starts at rest, swings out as
		// the part lands, and ends exactly where it started.
		swing = angle * (1 - t) * (1 - t) * float32(math.Sin(float64(t)*swayCycles*2*math.Pi))
	}

	// The figure is built whole and cut to the newest part's progress only
	// after it is roughened, so a hand-drawn line keeps its wobble while it
	// is drawn on instead of shimmering. The strokes from settled on are the
	// newest part's, and the face's when it arrives with that part.
	parts := gallows.FigureStrokes(budget, wrong, 1)
	fresh := len(parts)
	if drawn != settled {
		fresh = min(settled, len(parts))
	}
	look := func(s gallows.Stroke, seed int) gallows.Stroke {
		return s.Rough(uint64(seed)*7919+11, 1.4)
	}
	var still, hanging, drawing []gallows.Stroke
	for i, s := range gallows.FrameStrokes() {
		if s.Ink == gallows.Rope {
			hanging = append(hanging, look(s, i))
		} else {
			still = append(still, look(s, i))
		}
	}
	// The fresh strokes share the progress in turn, so several parts draw
	// on one after another; the face, after the last part, goes with it.
	partsFresh := max(drawn-settled, 1)
	for i, s := range parts {
		s = look(s, 100+i)
		if i >= fresh {
			turn := float32(min(i, drawn-1) - settled)
			own := min(max(progress*float32(partsFresh)-turn, 0), 1)
			s = s.Partial(own)
			if own < 1 && len(s.Points) > 0 {
				drawing = append(drawing, s)
			}
		}
		hanging = append(hanging, s)
	}

	box.Draw(func(pt *ui.Painter, r ui.Rect) {
		fit := gallows.Fit(r.W, r.H)
		if fit.Scale <= 0 {
			return
		}
		at := func(q gallows.Point) (float32, float32) {
			m := fit.Map(q)
			return r.X + m.X, r.Y + m.Y
		}
		path := func(points []gallows.Point) *ui.Path {
			var path ui.Path
			for i, q := range points {
				x, y := at(q)
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			return &path
		}
		stroke := func(s gallows.Stroke) {
			if len(s.Points) < 2 {
				return
			}
			width, color := fit.LineWidth(s.Width), colors[s.Ink]
			// A marker's line: thinner toward both ends, segment by segment,
			// with a faint second pass alongside it.
			n := len(s.Points)
			for j := 1; j < n; j++ {
				u := float32(j) / float32(n)
				taper := 0.55 + 0.45*min(1, u/0.15, (1-u)/0.15)
				pt.StrokePath(path(s.Points[j-1:j+1]), width*taper, color)
			}
			pt.StrokePath(path(s.Points), width*0.35, color.Alpha(0.35))
		}

		for _, s := range still {
			stroke(s)
		}
		px, py := at(gallows.Pivot())
		pt.Transform(ui.Rotation(swing), px, py, func() {
			for _, s := range hanging {
				stroke(s)
			}
			// The pen tip rides the end of each stroke still being drawn.
			if progress < 1 {
				for _, s := range drawing {
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

// gallowsLabel names the drawing for screen readers, which cannot see it.
func gallowsLabel(wrong, budget int) string {
	return fmt.Sprintf("The gallows, %d of %d wrong guesses", wrong, budget)
}
