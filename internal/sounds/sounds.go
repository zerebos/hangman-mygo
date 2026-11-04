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

// Palette is one set of the game's cues. Right, Hint and Wrong each play
// one note, at the step they are given (see Step): up the scale for right
// letters, down it for wrong ones.
type Palette struct {
	Name       string
	Right      func(step int) []float32
	Wrong      func(step int) []float32
	Hint       func(step int) []float32
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

// Top is the step a climb or a fall ends on: one octave of the pentatonic
// scale away from where it began. A word's last right letter lands on C6,
// the octave of the first and the note the win resolves to, so solving a
// word sounds like arriving rather than reaching.
const Top = len(pentatonic)

// Step is which note the k-th of n guesses plays, counting from 0, spread so
// the n-th always lands on Top: k counts right letters out of the word's
// distinct letters, or wrong ones out of the guess budget. One guess is one
// note, however many cells it fills.
func Step(k, n int) int {
	if n <= 1 {
		return Top
	}
	return int(math.Round(float64(min(max(k, 0), n-1)) * float64(Top) / float64(n-1)))
}

// minor is the minor pentatonic scale in semitones below its root, for the
// fall of the wrong letters.
var minor = [...]int{0, 2, 5, 7, 9}

// missRoot is the note the wrong letters fall from (A3): the key of the word
// lost phrase, so a lost word sounds like the end of a fall already heard.
const missRoot = 57

// missNote is the step-th note down the minor scale from missRoot.
func missNote(step int) int {
	step = max(step, 0)
	return missRoot - 12*(step/len(minor)) - minor[step%len(minor)]
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

// woodRight is a right letter: a pluck at the step-th note up the scale, so
// a word being solved is heard climbing to its octave.
func woodRight(step int) []float32 {
	b := newBuffer(0.9)
	b.mix(pluck(freq(scaleNote(step)), 0.9, 1), 0, 1)
	return b.samples(0.55)
}

// woodWrong is a wrong letter: the same wooden bar, struck low and damped at
// once, at the step-th note down a minor scale. It is one instrument with
// the right letters and the word lost, falling where they climb.
func woodWrong(step int) []float32 {
	f := freq(missNote(step))
	b := newBuffer(0.5)
	b.mix(pluck(f, 0.45, 0.35), 0, 1)
	// A little of the mallet's body, so it lands as a dead note rather than
	// a quiet one.
	b.mix(knock(f), 0, 0.35)
	return b.samples(0.45)
}

// woodHint is a letter bought with a guess: the right-letter pluck, with a
// little shimmer two octaves above it to mark it as given rather than found.
func woodHint(step int) []float32 {
	f := freq(scaleNote(step))
	b := newBuffer(0.9)
	b.mix(pluck(f, 0.9, 1), 0, 1)
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
