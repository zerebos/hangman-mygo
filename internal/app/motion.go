package app

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Everything here is feel only: no timing can change a rule, a score or the
// layout. Each effect is a one-shot Keyframes sequence whose last value is
// how the element looks at rest, and whose Run is a count that moves once per
// guess, so the next guess plays it again. Keyframes goes straight to its
// last value when the desktop asks for less motion.

const (
	shakeDuration = 320 * time.Millisecond
	// shakeDistance is how far the shake throws the word row at its widest.
	shakeDistance = 5
	// shakeCycles is a multiple of half a cycle, so the wave ends on zero.
	shakeCycles  = 2.5
	shakeSamples = 24

	pipPulse    = 600 * time.Millisecond
	pipHalo     = 12
	pipHaloFade = 0.6

	keySettle = 240 * time.Millisecond
	drawOn    = 520 * time.Millisecond
)

// shakeOffset is the word row's offset delta of the way through a shake: a
// sine damped to nothing, so it starts and ends at zero.
func shakeOffset(delta float32) float32 {
	return shakeDistance * (1 - delta) * float32(math.Sin(float64(delta*shakeCycles)*2*math.Pi))
}

// shakeFrames samples shakeOffset, because Keyframes plays straight segments.
var shakeFrames = func() []ui.Keyframe {
	frames := make([]ui.Keyframe, shakeSamples+1)
	for i := range frames {
		delta := float32(i) / shakeSamples
		frames[i] = ui.Keyframe{
			At:    time.Duration(float32(shakeDuration) * delta),
			Value: shakeOffset(delta),
			Ease:  ui.Linear,
		}
	}
	return frames
}()

// reveal is a staggered fade-in: letters resolving one after another, each
// rising the last few pixels into place.
type reveal struct {
	fade, step time.Duration
	rise       float32
}

// winReveal is the whole word resolving left to right, once, when it is won.
var winReveal = reveal{fade: 260 * time.Millisecond, step: 55 * time.Millisecond, rise: 7}

// play fades the glyph in as the index-th letter of the reveal. Only the
// paint moves, so the row measures the same the whole way through.
func (r reveal) play(glyph ui.Element, key string, run uint64, index int) {
	t := glyph.Keyframes(playKey{key, int(run)}, []ui.Keyframe{{Value: 0}, {At: r.fade, Value: 1, Ease: ui.EaseInOut}},
		ui.KeyframeOptions{Delay: r.step * time.Duration(index)})
	glyph.Opacity(t).Translate(0, r.rise*(1-t))
}

// once is a 0 to 1 sequence over d, played afresh whenever run changes.
//
// Each run is a sequence of its own, keyed by the run, rather than one
// sequence replayed through KeyframeOptions.Run. A replayed sequence starts
// from the value it last showed, 1 here, so it played 1 to 1; snapping it
// back to 0 after a millisecond still showed the finished value on the first
// frame, which drew each new body part whole for a frame before it drew on.
// A new sequence starts at its first frame, 0, on the frame it begins.
func once(e ui.Element, key string, run int, d time.Duration, ease ui.Easing) float32 {
	return e.Keyframes(playKey{key, run}, []ui.Keyframe{{Value: 0}, {At: d, Value: 1, Ease: ease}})
}

// run is a count that moves once per event within a word (wrong guesses,
// letters guessed) made unique for the whole session. Those counts start
// over with every word, and an element that stays from word to word, as the
// gallows does, would otherwise find a run of the same number already played
// and show its end at once.
func (a *App) run(n int) int {
	return a.deal*100 + n
}

// playKey names one play of a sequence: its name and the run it belongs to.
type playKey struct {
	name string
	run  int
}

// glideOn gives e the stage's glide, except while the stats drawer moves.
// Then the layout changes under the stage every frame, and every row chasing
// its own spring stuttered and let the keys drift apart, so the stage follows
// the drawer exactly instead. (c.DisableAnimations would do that too, but it
// also stops the drawer's own spring.)
func (a *App) glideOn(e ui.Element) ui.Element {
	if !a.drawerMoving {
		e.Transition(glide)
	}
	return e
}

// The layout transitions. Each element of the stage glides to where the
// layout puts it, at its size, so a row that changes height carries the
// rest of the stage with it smoothly. A spring rather than a curve, so a
// change that lands on its way turns smoothly instead of stopping dead.
var (
	glide = ui.ElementTransition{Animation: &ui.Snappy}

	// keysSwap folds the keyboard down and away as the result card takes
	// its place, quickly, so the card after it is not kept waiting. For the
	// next word it comes back at once rather than waiting for the card to
	// go: the keys are where a guess flies from, and a player typing the
	// moment the word is dealt would otherwise see letters fly out of a
	// keyboard that is not there yet.
	keysSwap = ui.ElementTransition{
		Animation: &ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut},
		Enter:     &ui.Motion{Y: 16, Scale: 0.97},
		Exit:      &ui.Motion{Y: 10, Scale: 0.94},
	}
	// cardSwap rises the result card into the keyboard's place once the
	// keyboard has gone, so the two never lie across each other. It leaves
	// fast, fading under the keyboard coming back.
	cardSwap = ui.ElementTransition{
		Animation:  &ui.Animation{Duration: 240 * time.Millisecond, Ease: ui.EaseOut},
		Enter:      &ui.Motion{Y: 24, Scale: 0.94},
		Exit:       &ui.Motion{Y: 6, Scale: 0.98},
		AfterExits: true,
	}
	// wordSwap slides the old word up and out as the next comes up from
	// below.
	wordSwap = ui.ElementTransition{
		Animation: &ui.Smooth,
		Enter:     &ui.Motion{Y: 14},
		Exit:      &ui.Motion{Y: -14},
	}
	// sceneSwap trades the stage for the match's summary and back: the one
	// going sinks back and fades as the other comes forward.
	sceneSwap = ui.ElementTransition{
		Animation: &ui.Smooth,
		Enter:     &ui.Motion{Scale: 1.04},
		Exit:      &ui.Motion{Scale: 0.96},
	}
	// clueReveal opens the clue's line and fades it in.
	clueReveal = ui.ElementTransition{
		Animation: &ui.Smooth,
		Enter:     &ui.Motion{Collapse: true, Y: -6},
		Exit:      &ui.Motion{Collapse: true},
	}
)

// letterMatch is the identity a letter shares between its key and where it
// lands, so the one appearing flies from the one gone. The deal is part of
// it, so a letter of the word just played never flies into the keyboard of
// the next, and n tells apart the flights of one letter to several cells.
type letterMatch struct {
	deal   int
	letter rune
	n      int
}

// flight is the transition of either end of a letter's n-th flight.
//
// The flights of a letter in several cells all leave at once, and the later
// ones take a little longer to arrive, so they still read as a stream. They
// are not delayed instead: a flight waiting to leave is held at its start,
// and its start is in the keyboard, which may be moving (folding away under
// the result card, or rising back for the next word), so a waiting letter
// rode along with it and only then turned for the word.
func (a *App) flight(letter rune, n int) ui.ElementTransition {
	anim := ui.Animation{Spring: ui.SpringOptions{Duration: flightTime + time.Duration(n)*flightStagger, Bounce: 0.15}}
	return ui.ElementTransition{Animation: &anim, Match: letterMatch{a.deal, letter, n}, Fly: true}
}

// flightTime is how long a letter's first flight takes, and flightStagger
// how much longer each later one of the same letter takes.
const (
	flightTime    = 420 * time.Millisecond
	flightStagger = 70 * time.Millisecond
)

// occurrences is how many times letter is in word.
func occurrences(word string, letter rune) int {
	n := 0
	for _, r := range word {
		if r == letter {
			n++
		}
	}
	return n
}

// pipPop swells a pip to nearly twice its size and settles it back with a
// little overshoot.
var pipPop = []ui.Keyframe{
	{Value: 1},
	{At: 110 * time.Millisecond, Value: 1.9, Ease: ui.EaseOut},
	{At: 300 * time.Millisecond, Value: 0.85, Ease: ui.EaseInOut},
	{At: 450 * time.Millisecond, Value: 1, Ease: ui.EaseInOut},
}
