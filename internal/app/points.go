package app

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// pointsFlight is the points of the word just won. They stay in the result
// card, and a copy of them flies from there into the Score chip. The chip
// shows the score without them until the copy lands, so it rolls up as they
// arrive rather than as the word is won.
type pointsFlight struct {
	// points is what the word earned, or 0 when nothing is on its way.
	points uint32
	// total is the match's score with the points in it.
	total uint32
	// deal is the word they were won on.
	deal int
	// start is when the card first showed them, or zero until a frame has.
	start time.Time
	// landed is whether the copy has reached the chip.
	landed bool
}

// pointsPhase is where the points of a won word are.
type pointsPhase int

const (
	// pointsNone means there are no points on their way.
	pointsNone pointsPhase = iota
	// pointsInCard means the copy sits in the result card, over the points
	// it is a copy of.
	pointsInCard
	// pointsFlying means the copy is on its way to the chip.
	pointsFlying
	// pointsLanded means the copy has reached the chip, and only the card
	// shows the points.
	pointsLanded
)

const (
	// pointsHold is how long the points sit in the card before they leave,
	// long enough to read them.
	pointsHold = 700 * time.Millisecond
	// pointsTravel is how long they take to reach the chip.
	pointsTravel = 600 * time.Millisecond
)

// phase is where the points are at now.
func (f pointsFlight) phase(now time.Time) pointsPhase {
	if f.points == 0 {
		return pointsNone
	}
	if f.landed {
		return pointsLanded
	}
	if f.start.IsZero() {
		return pointsInCard
	}
	switch t := now.Sub(f.start); {
	case t < pointsHold:
		return pointsInCard
	case t < pointsHold+pointsTravel:
		return pointsFlying
	}
	return pointsLanded
}

// label is the points as the card and the flight show them.
func (f pointsFlight) label() string { return "+" + strconv.FormatUint(uint64(f.points), 10) }

// pointsMatch is the identity the copy shares between the card and the
// chip, so the copy appearing at the chip flies from the one in the card.
type pointsMatch struct{ deal int }

// pointsInFlight is the transition of either end of the copy's flight. The
// end at the chip fades into it as it lands.
func (a *App) pointsInFlight() ui.ElementTransition {
	anim := ui.Animation{Spring: ui.SpringOptions{Duration: pointsTravel, Bounce: 0.1}}
	return ui.ElementTransition{
		Animation:    &anim,
		Match:        pointsMatch{a.points.deal},
		Fly:          true,
		ScaleContent: true,
		Exit:         &ui.Motion{Scale: 0.6},
	}
}

// advancePoints moves the copy along for this frame and says where it is. It asks for a frame at the next step, because nothing else on screen
// may be moving to bring one.
func (a *App) advancePoints(c *ui.Context) pointsPhase {
	f := &a.points
	if f.points == 0 {
		return pointsNone
	}
	// A new match starts its score over, so the last one's points have
	// nowhere to land, and its card is gone.
	if a.session.MatchPoints() != f.total {
		*f = pointsFlight{}
		return pointsNone
	}
	now := c.Now()
	if f.start.IsZero() {
		f.start = now
	}
	// With reduced motion nothing flies, so the chip has the points already.
	if c.Preferences().ReduceMotion {
		return pointsInCard
	}
	// The card leaving, for the next word or the match's summary, sends the
	// copy on its way at once instead of taking it along.
	if f.phase(now) == pointsInCard && (f.deal != a.deal || a.summaryShown) {
		f.start = now.Add(-pointsHold)
	}
	phase := f.phase(now)
	switch phase {
	case pointsInCard:
		c.After(f.start.Add(pointsHold).Sub(now))
	case pointsFlying:
		c.After(f.start.Add(pointsHold + pointsTravel).Sub(now))
	case pointsLanded:
		if !f.landed {
			f.landed = true
			a.landings++
		}
	}
	return phase
}

// shownScore is the match's score as the Score chip shows it: without the
// points whose copy is still on its way.
func (a *App) shownScore(phase pointsPhase, reduceMotion bool) uint32 {
	score := a.session.MatchPoints()
	if (phase == pointsInCard || phase == pointsFlying) && !reduceMotion {
		score -= a.points.points
	}
	return score
}

// chipPop swells a chip as something lands in it: the Score chip as the
// points arrive, and the Best chip as it takes a new record.
var chipPop = []ui.Keyframe{
	{Value: 1},
	{At: 90 * time.Millisecond, Value: 1.12, Ease: ui.EaseOut},
	{At: 320 * time.Millisecond, Value: 1, Ease: ui.EaseInOut},
}
