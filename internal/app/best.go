package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// newBestDelay is how long after the winning letter the Best chip takes the
// new record and pops, and the record's chime plays. It lands after the
// win's arpeggio has reached its chord, so the two are heard one after the
// other.
const newBestDelay = 500 * time.Millisecond

// bestMoment is a streak that has just beaten the best, waiting for its
// moment. The Best chip shows the old record until then.
type bestMoment struct {
	// from is the record that was beaten, and to the new one, or 0 when
	// there is no moment waiting.
	from, to uint32
	// start is when a frame first saw the moment, or zero until one has.
	start time.Time
}

// advanceBest moves the moment along for this frame and says whether the
// Best chip still holds the old record. It asks for a frame at the moment,
// because nothing else on screen may be moving to bring one.
func (a *App) advanceBest(c *ui.Context) (held bool) {
	m := &a.best
	if m.to == 0 {
		return false
	}
	// Reset stats throws the record away before its moment came.
	if a.session.Stats().BestStreak != m.to {
		*m = bestMoment{}
		return false
	}
	// With reduced motion the chip has the record at once, and does not
	// pop.
	if c.Preferences().ReduceMotion {
		*m = bestMoment{}
		return false
	}
	now := c.Now()
	if m.start.IsZero() {
		m.start = now
	}
	if wait := m.start.Add(newBestDelay).Sub(now); wait > 0 {
		c.After(wait)
		return true
	}
	*m = bestMoment{}
	a.bestPops++
	return false
}

// shownBest is the best streak as the Best chip shows it.
func (a *App) shownBest() uint32 {
	if a.bestHeld {
		return a.best.from
	}
	return a.session.Stats().BestStreak
}
