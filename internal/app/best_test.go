package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/sounds"
	"github.com/zerebos/hangman-mygo/internal/stats"
)

func TestOnlyBeatingABestThatWasThereIsANewBest(t *testing.T) {
	for _, tc := range []struct {
		before, after uint32
		want          bool
	}{
		{0, 1, false}, // the first win sets the first best
		{2, 2, false}, // a tie, or a win below the best
		{2, 3, true},
	} {
		if got := isNewBest(tc.before, tc.after); got != tc.want {
			t.Errorf("best %d to %d: %v", tc.before, tc.after, got)
		}
	}
}

// onAStreak is a tester whose next win makes a streak of streak+1 against a
// best of best, with a recorder for the cues.
func onAStreak(streak, best uint32) (*App, *ui.Tester, *recorder) {
	cues := &recorder{}
	a := New(settings.Settings{Stats: stats.Stats{Streak: streak, BestStreak: best}}, Options{Speaker: cues})
	return a, ui.NewTester(a.View, 1000, 800), cues
}

// The Best chip holds the beaten record a moment after the win, then takes
// the new one and pops, with a chime, and asks for no frames after.
func TestANewBestPopsTheBestChipWithAChime(t *testing.T) {
	a, tt, cues := onAStreak(2, 2)
	winWord(a, tt)
	if !cues.heard(sounds.Wood.NewBest()) {
		t.Error("the new best played no chime")
	}
	tt.Advance(newBestDelay / 2)
	if !a.bestHeld || a.shownBest() != 2 || a.bestPops != 0 {
		t.Errorf("before its moment: held %v, shown %d, pops %d", a.bestHeld, a.shownBest(), a.bestPops)
	}
	tt.Advance(newBestDelay)
	if a.bestHeld || a.shownBest() != 3 || a.bestPops != 1 {
		t.Errorf("after its moment: held %v, shown %d, pops %d", a.bestHeld, a.shownBest(), a.bestPops)
	}
	tt.Advance(3 * time.Second)
	if tt.FramePending() || a.bestPops != 1 {
		t.Errorf("frames still asked for %v, pops %d", tt.FramePending(), a.bestPops)
	}
}

// A win that only ties the best, or sets the very first one, is no record.
func TestATieOrAFirstBestIsNoMoment(t *testing.T) {
	for _, tc := range []struct{ streak, best uint32 }{{1, 2}, {0, 0}} {
		a, tt, cues := onAStreak(tc.streak, tc.best)
		winWord(a, tt)
		tt.Advance(2 * newBestDelay)
		if cues.heard(sounds.Wood.NewBest()) || a.bestPops != 0 {
			t.Errorf("streak %d, best %d: chimed %v, pops %d", tc.streak, tc.best, cues.heard(sounds.Wood.NewBest()), a.bestPops)
		}
	}
}

// With reduced motion the chip shows the record at once and does not pop,
// but the chime still plays, because a sound is not motion.
func TestReducedMotionShowsTheNewBestAtOnce(t *testing.T) {
	a, tt, cues := onAStreak(2, 2)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	winWord(a, tt)
	if a.bestHeld || a.shownBest() != 3 || a.bestPops != 0 || !cues.heard(sounds.Wood.NewBest()) {
		t.Errorf("held %v, shown %d, pops %d, chimed %v", a.bestHeld, a.shownBest(), a.bestPops, cues.heard(sounds.Wood.NewBest()))
	}
}

// Resetting the stats before the moment drops it, rather than popping a
// record that is gone.
func TestResettingTheStatsDropsTheMoment(t *testing.T) {
	a, tt, _ := onAStreak(2, 2)
	winWord(a, tt)
	a.resetStats()
	tt.Advance(2 * newBestDelay)
	if a.bestPops != 0 || a.shownBest() != 0 {
		t.Errorf("pops %d, shown %d", a.bestPops, a.shownBest())
	}
}
