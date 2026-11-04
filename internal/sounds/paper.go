package sounds

import (
	"math"
	"math/rand/v2"
)

// Paper is the pencil-and-paper palette: no notes, only the sounds of
// writing, to match the hand-drawn gallows.
var Paper = Palette{
	Name: "paper", Right: paperRight, Wrong: paperWrong, Hint: paperHint,
	WordWon: paperWordWon, WordLost: paperWordLost, MatchWon: paperMatchWon, MatchEnded: paperMatchEnded,
}

// bandpass is a two-pole resonant filter (the RBJ cookbook's band-pass with
// constant peak gain), whose centre can move from sample to sample, as a
// pencil's pitch rises and falls with its speed.
type bandpass struct {
	q              float64
	x1, x2, y1, y2 float64
}

func (f *bandpass) step(x, centre float64) float64 {
	w := 2 * math.Pi * centre / Rate
	alpha := math.Sin(w) / (2 * f.q)
	a0 := 1 + alpha
	b0, b2 := alpha/a0, -alpha/a0
	a1, a2 := -2*math.Cos(w)/a0, (1-alpha)/a0
	y := b0*x + b2*f.x2 - a1*f.y1 - a2*f.y2
	f.x2, f.x1 = f.x1, x
	f.y2, f.y1 = f.y1, y
	return y
}

// stroke is a pencil drawn across paper for seconds: noise through a band
// whose centre slides from one frequency to another as the stroke speeds up
// or slows down, its loudness wavering with the paper's grain, easing in and
// out at its ends.
func stroke(seconds, from, to float64, seed uint64) buffer {
	b := newBuffer(seconds)
	rng := rand.New(rand.NewPCG(seed, 21))
	f := bandpass{q: 1.4}
	grain := 0.0
	for i := range b {
		t := float64(i) / float64(len(b))
		centre := from * math.Pow(to/from, t)
		// The paper's tooth: a slow random flutter in loudness.
		grain += 0.004 * (rng.Float64()*2 - 1 - grain)
		env := math.Sin(math.Pi*math.Min(t*4, 1)/2) * math.Sin(math.Pi*math.Min((1-t)*5, 1)/2)
		b[i] = f.step(rng.Float64()*2-1, centre) * env * (0.7 + 6*grain)
	}
	return b
}

// tap is the pencil's point set down on the paper: a click ringing briefly
// at a pitch, with a soft low thump of the desk under it.
func tap(pitch float64, seed uint64) buffer {
	b := newBuffer(0.08)
	rng := rand.New(rand.NewPCG(seed, 33))
	f := bandpass{q: 6}
	for i := range b {
		t := float64(i) / Rate
		x := 0.0
		if t < 0.0015 {
			x = rng.Float64()*2 - 1
		}
		ring := f.step(x, pitch) * 6
		thump := math.Sin(2*math.Pi*180*t) * math.Exp(-t/0.008)
		// A millisecond's attack, so the point lands rather than pops.
		b[i] = (ring*math.Exp(-t/0.018) + 0.35*thump) * math.Min(t/0.001, 1)
	}
	return b
}

// crumple is paper being balled up: bursts of crackle, thick in the middle
// and thinning out, each a click through a bright band.
func crumple(seconds float64, seed uint64) buffer {
	b := newBuffer(seconds)
	rng := rand.New(rand.NewPCG(seed, 44))
	f := bandpass{q: 0.9}
	impulse := 0.0
	for i := range b {
		t := float64(i) / float64(len(b))
		density := 0.004 * math.Sin(math.Pi*t) * (1.2 - t)
		if rng.Float64() < density {
			impulse = (rng.Float64()*2 - 1) * (0.4 + rng.Float64())
		} else {
			impulse *= 0.6
		}
		b[i] = f.step(impulse, 2600+1800*rng.Float64())
	}
	return b
}

// check is a check mark: a short stroke down and a longer one up and away.
func check(seed uint64) buffer {
	b := newBuffer(0.24)
	b.mix(stroke(0.07, 1500, 1100, seed), 0, 0.8)
	b.mix(stroke(0.13, 1300, 2600, seed+1), 0.075, 1)
	return b
}

// paperRight is a right letter: the pencil tapped down, a little higher for
// each right letter so far so progress is still heard, then a check mark. A
// letter in several cells taps once for each, a beat apart.
func paperRight(step, count int) []float32 {
	count = max(count, 1)
	const apart = 0.045
	pitch := 1700 * math.Pow(2, float64(min(step, 12))/12)
	b := newBuffer(0.12 + apart*float64(count-1) + 0.26)
	for n := range count {
		b.mix(tap(pitch, uint64(n)), apart*float64(n), 1)
	}
	b.mix(check(uint64(step)), 0.06+apart*float64(count-1), 0.7)
	return b.samples(0.5)
}

// paperWrong is a wrong letter: the pen drawing the new body part, a longer
// stroke that drags, low in the band.
func paperWrong() []float32 {
	b := newBuffer(0.56)
	b.mix(tap(900, 5), 0, 0.6)
	b.mix(stroke(0.42, 700, 1000, 7), 0.03, 1)
	return b.samples(0.55)
}

// paperHint is a letter bought with a guess: a flick of the page before the
// tap and the check, to mark it as given rather than found.
func paperHint(step, count int) []float32 {
	right := paperRight(step, count)
	b := newBuffer(0.14 + float64(len(right))/Rate)
	b.mix(stroke(0.12, 3500, 5500, 13), 0, 0.5)
	for i, v := range right {
		b[i+int(0.1*Rate)] += float64(v)
	}
	return b.samples(0.5)
}

// paperWordWon is a word solved: a quick flourish underlining it, ending on
// a full stop.
func paperWordWon() []float32 {
	b := newBuffer(0.75)
	b.mix(stroke(0.16, 1200, 2200, 31), 0, 0.8)
	b.mix(stroke(0.34, 1600, 3000, 32), 0.15, 1)
	b.mix(tap(2400, 9), 0.56, 1.2)
	return b.samples(0.6)
}

// paperWordLost is a word lost: the page balled up.
func paperWordLost() []float32 {
	b := newBuffer(0.75)
	b.mix(crumple(0.7, 17), 0, 1)
	return b.samples(0.5)
}

// paperMatchWon is a match won, with the confetti: three quick ticks down a
// list and a long underline under them.
func paperMatchWon() []float32 {
	b := newBuffer(1.5)
	for n := range 3 {
		b.mix(check(uint64(40+n)), 0.2*float64(n), 0.8)
	}
	b.mix(stroke(0.55, 1200, 2800, 50), 0.68, 1)
	b.mix(tap(2400, 51), 1.28, 1.1)
	return b.samples(0.65)
}

// paperMatchEnded is a match lost or tied: the page turned over.
func paperMatchEnded() []float32 {
	b := newBuffer(0.8)
	b.mix(stroke(0.6, 500, 3200, 60), 0, 1)
	b.mix(tap(600, 61), 0.58, 0.7)
	return b.samples(0.5)
}
