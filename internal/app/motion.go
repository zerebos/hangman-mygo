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

	pipPulse    = 420 * time.Millisecond
	pipHalo     = 7
	pipHaloFade = 0.45

	keySettle = 240 * time.Millisecond
	drawOn    = 300 * time.Millisecond
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

// guessReveal is the cells one correct guess turned over: quicker and
// tighter, because it is feedback on one guess rather than the end.
var guessReveal = reveal{fade: 200 * time.Millisecond, step: 70 * time.Millisecond, rise: 5}

// play fades the glyph in as the index-th letter of the reveal. Only the
// paint moves, so the row measures the same the whole way through.
func (r reveal) play(glyph ui.Element, key string, run uint64, index int) {
	t := glyph.Keyframes(key, []ui.Keyframe{{Value: 0}, {At: r.fade, Value: 1, Ease: ui.EaseInOut}},
		ui.KeyframeOptions{Run: run, Delay: r.step * time.Duration(index)})
	glyph.Opacity(t).Translate(0, r.rise*(1-t))
}

// once is a 0 to 1 sequence over d, replayed whenever run changes.
func once(e ui.Element, key string, run int, d time.Duration, ease ui.Easing) float32 {
	return e.Keyframes(key, []ui.Keyframe{{Value: 0}, {At: d, Value: 1, Ease: ease}},
		ui.KeyframeOptions{Run: uint64(run)})
}

// The layout transitions. Each element of the stage glides to where the
// layout puts it, at its size, so a row that changes height carries the
// rest of the stage with it smoothly. A spring rather than a curve, so a
// change that lands on its way turns smoothly instead of stopping dead.
var (
	glide = ui.ElementTransition{Animation: &ui.Snappy}

	// keysSwap folds the keyboard down and away as the result card takes
	// its place, and brings it back up for the next word. It is quick, so
	// the card after it is not kept waiting.
	keysSwap = ui.ElementTransition{
		Animation:  &ui.Animation{Duration: 180 * time.Millisecond, Ease: ui.EaseOut},
		Enter:      &ui.Motion{Y: 16, Scale: 0.97},
		Exit:       &ui.Motion{Y: 10, Scale: 0.94},
		AfterExits: true,
	}
	// cardSwap rises the result card into the keyboard's place once the
	// keyboard has gone, so the two never lie across each other.
	cardSwap = ui.ElementTransition{
		Animation:  &ui.Snappy,
		Enter:      &ui.Motion{Y: 24, Scale: 0.94},
		Exit:       &ui.Motion{Y: 10, Scale: 0.96},
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
// the next.
type letterMatch struct {
	deal   int
	letter rune
}

// flight is the transition of either end of a letter's flight.
func flight(deal int, letter rune) ui.ElementTransition {
	return ui.ElementTransition{Animation: &ui.Snappy, Match: letterMatch{deal, letter}, Fly: true}
}
