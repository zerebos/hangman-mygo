package app

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo/ui"
)

// confettiLife is how long a burst is in the air, fading over its last
// third.
const confettiLife = 2600 * time.Millisecond

// gravity pulls the pieces down, in DIPs per second squared.
const gravity = 900

var confettiColors = []ui.Color{
	ui.Hex("#26ccff"), ui.Hex("#a25afd"), ui.Hex("#ff5e7e"), ui.Hex("#88ff5a"),
	ui.Hex("#fcff42"), ui.Hex("#ffa62d"), ui.Hex("#ff36ff"),
}

// piece is one piece of confetti: where it is thrown from, as fractions of
// the window, how fast, and how it looks and turns.
type piece struct {
	x, y   float32
	vx, vy float32 // DIPs per second
	w, h   float32
	spin   float32 // degrees per second
	color  ui.Color
}

// newConfetti throws two bursts up and inwards from the lower corners.
func newConfetti() []piece {
	pieces := make([]piece, 0, 120)
	for side := range 2 {
		x, dir := float32(0.08), float32(1)
		if side == 1 {
			x, dir = 0.92, -1
		}
		for range 60 {
			angle := (55 + 25*rand.Float64()) * math.Pi / 180
			speed := 700 + 500*rand.Float64()
			pieces = append(pieces, piece{
				x: x, y: 0.95,
				vx:    dir * float32(math.Cos(angle)*speed),
				vy:    -float32(math.Sin(angle) * speed),
				w:     8 + 4*rand.Float32(),
				h:     5 + 4*rand.Float32(),
				spin:  (180 + 360*rand.Float32()) * float32(1-2*rand.IntN(2)),
				color: confettiColors[rand.IntN(len(confettiColors))],
			})
		}
	}
	return pieces
}

// confetti draws the pieces over the whole window, t of the way through
// their flight. It takes no input, so the summary under it stays usable.
func confetti(c *ui.Context, pieces []piece, run int) {
	layer := ui.Box(c.Key("confetti")).Absolute().Top(0).Left(0).Right(0).Bottom(0).IgnoreInput(true)
	t := once(layer, "flight", run, confettiLife, ui.Linear)
	if t >= 1 {
		return
	}
	layer.Draw(func(p *ui.Painter, r ui.Rect) {
		secs := t * float32(confettiLife.Seconds())
		fade := min(1, (1-t)*3)
		for _, pc := range pieces {
			x := r.X + pc.x*r.W + pc.vx*secs
			y := r.Y + pc.y*r.H + pc.vy*secs + gravity/2*secs*secs
			// Air drag, roughly: the pieces slow as they go, so they hang at
			// the top of their arc instead of shooting off the window.
			x -= pc.vx * secs * secs * 0.25
			p.Transform(ui.Rotation(pc.spin*secs), x, y, func() {
				p.Fill(ui.Rect{X: x - pc.w/2, Y: y - pc.h/2, W: pc.w, H: pc.h}, pc.color.Alpha(fade), 1)
			})
		}
	})
}
