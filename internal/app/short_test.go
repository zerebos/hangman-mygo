package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestTheBoardTakesAShapeThatFitsThePage(t *testing.T) {
	for _, tc := range []struct {
		w, h float32
		want boardShape
	}{
		{1000, 800, boardStacked},
		{880, 660, boardStacked}, // the desktop's smallest window
		{390, 844, boardStacked}, // a phone upright
		{844, 390, boardSideBySide},
		{sideBySideW, shortH - 1, boardSideBySide},
		{sideBySideW - 1, shortH - 1, boardScrolled},
	} {
		if got := boardShapeFor(tc.w, tc.h); got != tc.want {
			t.Errorf("%v by %v: %v, want %v", tc.w, tc.h, got, tc.want)
		}
	}
}

func TestTheGallowsColumnStaysWithinItsBounds(t *testing.T) {
	for _, w := range []float32{sideBySideW, 667, 844, 2000} {
		gallows, room := sideColumns(w)
		if gallows < minGallowsColW || gallows > maxGallowsColW {
			t.Errorf("%v px: a gallows column of %v", w, gallows)
		}
		if total := gallows + room + sideGap + 2*stagePad; total > w+0.01 {
			t.Errorf("%v px: the columns take %v", w, total)
		}
	}
}

// fits reports whether the text s is on screen, whole.
func fits(t *testing.T, tt *ui.Tester, w, h int, s string) {
	t.Helper()
	r, ok := tt.Find(s)
	if !ok {
		t.Errorf("%dx%d: no %q", w, h, s)
		return
	}
	if r.X < 0 || r.Y < 0 || r.X+r.W > float32(w)+0.5 || r.Y+r.H > float32(h)+0.5 {
		t.Errorf("%dx%d: %q at %+v runs off the page", w, h, s, r)
	}
}

// A phone held sideways fits the whole board, the actions under the keys
// included, and the result card once the word is won. On the narrowest of
// them the chips stay in the bar, and the wordmark makes room.
func TestAPhoneOnItsSideFitsTheWholeBoard(t *testing.T) {
	for _, size := range [][2]int{{844, 390}, {667, 375}, {sideBySideW, 375}} {
		w, h := size[0], size[1]
		a, tt := touchTester(true)
		tt.SetSize(w, h)
		tt.Advance(2 * time.Second)
		for _, s := range []string{"A", "Z", "Hint", "Give up", "Score", "Word"} {
			fits(t, tt, w, h, s)
		}
		_, wordmark := tt.Find("Hangman!")
		if wordmark != (w >= chipsBelowW) {
			t.Errorf("%dx%d: wordmark shown %v", w, h, wordmark)
		}
		for _, r := range a.game.Word() {
			if 'A' <= r && r <= 'Z' {
				a.guess(r)
			}
		}
		tt.Frame()
		tt.Advance(3 * time.Second)
		fits(t, tt, w, h, "New Game")
	}
}

// The drawer covers a short page rather than taking room its board needs.
func TestTheDrawerCoversAShortPage(t *testing.T) {
	_, _, before, after := openStats(t, 1000, 500)
	if before != after {
		t.Errorf("the A key moved from %v to %v under the drawer", before, after)
	}
}
