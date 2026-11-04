// Package sounds synthesizes the game's audio cues as samples, with no audio
// device and no mygo in it, so the cues can be tested like any other code.
// The game hands the samples to mygo's audio package to play.
//
// The palette, Wood, is a soft wooden pluck, like a marimba bar, on a
// pentatonic scale, so any notes played together or in a row agree; a wrong
// guess is a muted knock under a short marker scratch. A pencil-and-paper
// palette was tried beside it and lost; it is in commit 3dcc085.
package sounds

import (
	"math"
	"math/rand/v2"
)

// Palette is one set of the game's cues. Right and Hint play one note for
// each cell the letter fills, at the steps up the scale they are given (see
// Steps).
type Palette struct {
	Name       string
	Right      func(steps ...int) []float32
	Wrong      func() []float32
	Hint       func(steps ...int) []float32
	WordWon    func() []float32
	WordLost   func() []float32
	MatchWon   func() []float32
	MatchEnded func() []float32
}

// Wood is the marimba palette.
var Wood = Palette{
	Name: "wood", Right: woodRight, Wrong: woodWrong, Hint: woodHint,
	WordWon: woodWordWon, WordLost: woodWordLost, MatchWon: woodMatchWon, MatchEnded: woodMatchEnded,
}

// Rate is the sample rate of every cue: mygo's output rate, so nothing is
// resampled on the way out.
const Rate = 48000

// Every cue is mono.

// pentatonic is the major pentatonic scale in semitones above its root. Any
// of its notes sound well together, which matters when plucks overlap.
var pentatonic = [...]int{0, 2, 4, 7, 9}

// root is the note the right-letter climb starts on, as a MIDI number (C5).
const root = 72

// freq is the frequency of a MIDI note.
func freq(note int) float64 {
	return 440 * math.Pow(2, float64(note-69)/12)
}

// Top is the highest step a word climbs to: two octaves of the pentatonic
// scale.
const Top = 2*len(pentatonic) - 1

// Steps is the notes a guess plays: one for each of the count cells it
// filled, after filled cells of the word's cells were already showing, spread
// over the scale so the word's last cell always lands on Top. A word being
// solved is heard climbing to its end, and a letter in two cells plays a
// little run rather than one note twice, which only blurs.
func Steps(filled, count, cells int) []int {
	steps := make([]int, max(count, 1))
	for n := range steps {
		ordinal := filled + n
		if cells <= 1 {
			steps[n] = Top
			continue
		}
		steps[n] = int(math.Round(float64(ordinal) * float64(Top) / float64(cells-1)))
	}
	return steps
}

// scaleNote is the step-th note of the pentatonic scale above root, climbing
// through octaves.
func scaleNote(step int) int {
	step = max(step, 0)
	return root + 12*(step/len(pentatonic)) + pentatonic[step%len(pentatonic)]
}

// buffer is a cue being built, in float64 for headroom while it is mixed.
type buffer []float64

func newBuffer(seconds float64) buffer {
	return make(buffer, int(seconds*Rate))
}

// mix adds src into b starting at seconds.
func (b buffer) mix(src buffer, at, gain float64) {
	start := int(at * Rate)
	for i, v := range src {
		if j := start + i; j >= 0 && j < len(b) {
			b[j] += v * gain
		}
	}
}

// samples is the cue at peak, faded out over its last 20 ms so it never ends
// on a click.
func (b buffer) samples(peak float64) []float32 {
	top := 0.0
	for _, v := range b {
		top = max(top, math.Abs(v))
	}
	scale := 0.0
	if top > 0 {
		scale = peak / top
	}
	fade := Rate / 50
	out := make([]float32, len(b))
	for i, v := range b {
		if left := len(b) - i; left < fade {
			v *= float64(left) / float64(fade)
		}
		out[i] = float32(v * scale)
	}
	return out
}

// pluck is a struck wooden bar: a fundamental with the inharmonic overtones
// a marimba bar has (about 4 and 10 times its pitch), the higher ones dying
// faster, after a 1.5 ms attack and a tick of noise for the mallet.
func pluck(f, seconds, brightness float64) buffer {
	b := newBuffer(seconds)
	// The fundamental dies away over the pluck's length, to about 2% by its
	// end, so the cue is over when it stops rather than cut off ringing.
	partials := []struct{ ratio, amp, decay float64 }{
		{1, 1, seconds / 4},
		{3.93, 0.32 * brightness, min(0.12, seconds/8)},
		{9.8, 0.1 * brightness, 0.04},
	}
	for _, p := range partials {
		fp := f * p.ratio
		if fp > Rate*0.45 {
			continue
		}
		// A rotation for the sine and a product for the decay, which sin
		// and exp for every sample would make slow.
		w := 2 * math.Pi * fp / Rate
		cw, sw := math.Cos(w), math.Sin(w)
		s, c := 0.0, 1.0
		env, d := p.amp, math.Exp(-1/(p.decay*Rate))
		for i := range b {
			b[i] += env * s
			s, c = s*cw+c*sw, c*cw-s*sw
			env *= d
		}
	}
	rng := rand.New(rand.NewPCG(uint64(f), 7))
	lp := 0.0
	for i := range min(len(b), Rate/100) {
		t := float64(i) / Rate
		lp += 0.25 * (rng.Float64()*2 - 1 - lp)
		b[i] += 0.15 * lp * math.Exp(-t/0.002)
	}
	for i := range min(len(b), Rate*3/2000) {
		b[i] *= float64(i) / (Rate * 1.5 / 1000)
	}
	return b
}

// knock is a muted thud: a low sine whose pitch drops as it dies, as a soft
// mallet on a wooden box sounds, with a little low noise for the body.
func knock(f float64) buffer {
	b := newBuffer(0.25)
	rng := rand.New(rand.NewPCG(3, 9))
	phase, lp := 0.0, 0.0
	for i := range b {
		t := float64(i) / Rate
		// A small pitch drop and a 6 ms attack: a felt mallet rather than a
		// thump, so a sound heard on every miss does not grate.
		pitch := f * (1 + 0.25*math.Exp(-t/0.025))
		phase += 2 * math.Pi * pitch / Rate
		lp += 0.02 * (rng.Float64()*2 - 1 - lp)
		b[i] = (math.Sin(phase) + 0.8*lp) * math.Exp(-t/0.07) * min(t/0.006, 1)
	}
	return b
}

// scratch is a marker dragged across paper: noise kept to the middle of the
// range, swelling in and out over its length, with a little grain.
func scratch(seconds float64) buffer {
	b := newBuffer(seconds)
	rng := rand.New(rand.NewPCG(11, 5))
	lo, hi := 0.0, 0.0
	for i := range b {
		t := float64(i) / seconds / Rate
		x := rng.Float64()*2 - 1
		hi += 0.5 * (x - hi)   // keep the hiss below the top
		lo += 0.02 * (hi - lo) // and take the rumble out
		grain := 0.75 + 0.25*math.Sin(2*math.Pi*float64(i)/Rate*37)
		b[i] = (hi - lo) * math.Sin(math.Pi*t) * grain
	}
	return b
}

// woodRight is a right letter: the step-th pluck up the scale, one step for each
// right letter of the word so far, so a word being solved is heard climbing.
// A letter filling count cells plucks count times, a beat apart, as its
// letters fly into the word one after another.
func woodRight(steps ...int) []float32 {
	if len(steps) == 0 {
		steps = []int{0}
	}
	b := newBuffer(0.9 + noteApart*float64(len(steps)-1))
	for n, step := range steps {
		b.mix(pluck(freq(scaleNote(step)), 0.9, 1), noteApart*float64(n), 1)
	}
	return b.samples(0.55)
}

// noteApart is how far apart the notes of a letter in several cells sound,
// in step with their flights.
const noteApart = 0.045

// woodWrong is a wrong letter: a knock, with the scratch of the pen drawing the
// new part on top of it.
func woodWrong() []float32 {
	b := newBuffer(0.55)
	b.mix(knock(98), 0, 1)
	b.mix(scratch(0.45), 0.03, 0.18)
	return b.samples(0.42)
}

// woodHint is a letter bought with a guess: the right-letter pluck, with a
// little shimmer two octaves above it to mark it as given rather than found.
func woodHint(steps ...int) []float32 {
	if len(steps) == 0 {
		steps = []int{0}
	}
	f := freq(scaleNote(steps[len(steps)-1]))
	right := woodRight(steps...)
	b := newBuffer(max(float64(len(right))/Rate, 0.7))
	for i, v := range right {
		b[i] += float64(v)
	}
	b.mix(pluck(f*4, 0.6, 0.2), 0.03, 0.25)
	b.mix(pluck(f*4*1.5, 0.6, 0.2), 0.09, 0.18)
	return b.samples(0.55)
}

// phrase plucks notes one after another, gap seconds apart, and lets the
// last ring.
func phrase(notes []int, gap, ring float64) buffer {
	b := newBuffer(gap*float64(len(notes)) + ring)
	for i, note := range notes {
		b.mix(pluck(freq(note), ring, 1), gap*float64(i), 1)
	}
	return b
}

// chord plucks notes together, the lower ones a touch earlier, as a strum.
func chord(notes []int, at, ring float64, b buffer) {
	for i, note := range notes {
		b.mix(pluck(freq(note), ring, 0.8), at+0.012*float64(i), 0.8)
	}
}

// woodWordWon is a word solved: a quick rising arpeggio that lands on a chord.
func woodWordWon() []float32 {
	b := phrase([]int{72, 76, 79}, 0.07, 1.2)
	chord([]int{72, 76, 79, 84}, 0.21, 1.4, b)
	return b.samples(0.7)
}

// woodWordLost is a word lost: three soft notes stepping down in a minor key,
// slower than the win, sad rather than comic.
func woodWordLost() []float32 {
	b := newBuffer(1.6)
	for i, note := range []int{64, 60, 57} {
		b.mix(pluck(freq(note), 1.1, 0.5), 0.22*float64(i), 0.9-0.15*float64(i))
	}
	return b.samples(0.55)
}

// woodMatchWon is a match won, played with the confetti: a longer phrase up the
// scale that ends on a held chord.
func woodMatchWon() []float32 {
	b := phrase([]int{67, 72, 76, 79, 84}, 0.09, 1.0)
	b2 := newBuffer(float64(len(b))/Rate + 1.8)
	b2.mix(b, 0, 1)
	chord([]int{60, 67, 72, 76, 79}, 0.5, 1.8, b2)
	return b2.samples(0.75)
}

// woodMatchEnded is a match lost or tied: a gentle close, two notes coming home.
func woodMatchEnded() []float32 {
	b := newBuffer(1.8)
	b.mix(pluck(freq(67), 1.2, 0.6), 0, 0.9)
	chord([]int{60, 64, 67}, 0.3, 1.5, b)
	return b.samples(0.55)
}
