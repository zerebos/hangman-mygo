package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/settings"
)

func TestThePointsSitInTheCardThenFlyThenLand(t *testing.T) {
	start := time.Unix(1000, 0)
	f := pointsFlight{points: 140, total: 140}
	if got := f.phase(start); got != pointsInCard {
		t.Errorf("before a frame has seen them: %v", got)
	}
	f.start = start
	for _, tc := range []struct {
		after time.Duration
		want  pointsPhase
	}{
		{0, pointsInCard},
		{pointsHold - time.Millisecond, pointsInCard},
		{pointsHold, pointsFlying},
		{pointsHold + pointsTravel - time.Millisecond, pointsFlying},
		{pointsHold + pointsTravel, pointsLanded},
	} {
		if got := f.phase(start.Add(tc.after)); got != tc.want {
			t.Errorf("after %v: %v, want %v", tc.after, got, tc.want)
		}
	}
	if got := (pointsFlight{}).phase(start); got != pointsNone {
		t.Errorf("no points: %v", got)
	}
}

// shows reports whether the view has a text that is exactly s. HasText
// would find the score's "150" inside the points' "+150".
func shows(tt *ui.Tester, s string) bool {
	for _, text := range tt.Texts() {
		if text == s {
			return true
		}
	}
	return false
}

// winWord guesses every letter of the word on the board.
func winWord(a *App, tt *ui.Tester) {
	for _, r := range a.game.Word() {
		if 'A' <= r && r <= 'Z' && !a.game.Guessed().Has(r) {
			tt.Key(0, ui.KeyA+ui.Key(r-'A'))
		}
	}
}

// The Score chip keeps the old score while the points are in the card and
// on their way, takes them as they land, and asks for no frames after.
func TestTheScoreTakesTheWordsPointsAsTheyLand(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	winWord(a, tt)
	total := a.session.MatchPoints()
	label := a.points.label()
	if total == 0 || label != fmt.Sprintf("+%d", total) {
		t.Fatalf("score %d, points %q", total, label)
	}
	tt.Advance(pointsHold / 2)
	if !shows(tt, label) || shows(tt, fmt.Sprint(total)) {
		t.Errorf("in the card: texts %q", tt.Texts())
	}
	tt.Advance(pointsHold/2 + pointsTravel/2)
	if a.pointsNow != pointsFlying || shows(tt, fmt.Sprint(total)) {
		t.Errorf("on the way: phase %v, texts %q", a.pointsNow, tt.Texts())
	}
	tt.Advance(pointsTravel)
	if a.pointsNow != pointsLanded || a.landings != 1 || !shows(tt, fmt.Sprint(total)) {
		t.Errorf("landed: phase %v, landings %d, texts %q", a.pointsNow, a.landings, tt.Texts())
	}
	tt.Advance(2 * time.Second)
	if tt.FramePending() {
		t.Error("the chip kept asking for frames once the points had landed")
	}
	if a.landings != 1 {
		t.Errorf("the chip popped %d times for one word", a.landings)
	}
}

// The card keeps showing what the word earned after its copy has flown, so
// the points are still there to read.
func TestTheCardKeepsThePointsAfterTheyFly(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	winWord(a, tt)
	label := a.points.label()
	tt.Advance(pointsHold + pointsTravel/2)
	if n := countText(tt, label); n != 2 {
		t.Errorf("on the way: %d copies of %q, want the card's and the flight's", n, label)
	}
	tt.Advance(pointsTravel + time.Second)
	if n := countText(tt, label); n != 1 {
		t.Errorf("landed: %d copies of %q, want the card's", n, label)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(2 * time.Second)
	if shows(tt, label) {
		t.Error("the next word's board still shows the last word's points")
	}
}

// countText is how many of the view's texts are exactly s.
func countText(tt *ui.Tester, s string) int {
	n := 0
	for _, text := range tt.Texts() {
		if text == s {
			n++
		}
	}
	return n
}

// Enter during the hold deals the next word, and the points leave the card
// at once instead of leaving with it.
func TestTheNextWordSendsThePointsOnTheirWay(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	winWord(a, tt)
	tt.Advance(100 * time.Millisecond)
	tt.Key(0, ui.KeyEnter)
	if a.pointsNow != pointsFlying {
		t.Errorf("after Enter the points are %v", a.pointsNow)
	}
	tt.Advance(pointsTravel + 100*time.Millisecond)
	if a.pointsNow != pointsLanded || a.landings != 1 {
		t.Errorf("phase %v, landings %d", a.pointsNow, a.landings)
	}
}

// A new match starts the score over, so the points of the last one are
// dropped rather than taken off a score that no longer has them.
func TestANewMatchDropsThePointsOnTheirWay(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	winWord(a, tt)
	a.setDifficulty(a.game.Difficulty() + 1)
	tt.Frame()
	if a.points.points != 0 || a.pointsNow != pointsNone {
		t.Errorf("points %+v, phase %v", a.points, a.pointsNow)
	}
	if got := a.shownScore(a.pointsNow, false); got != 0 {
		t.Errorf("the new match shows a score of %d", got)
	}
}

// With reduced motion nothing flies: the points stay in the card and the
// chip has them at once.
func TestReducedMotionKeepsThePointsInTheCard(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	winWord(a, tt)
	total := a.session.MatchPoints()
	tt.Advance(3 * time.Second)
	if !shows(tt, a.points.label()) || !shows(tt, fmt.Sprint(total)) {
		t.Errorf("texts %q", tt.Texts())
	}
}
