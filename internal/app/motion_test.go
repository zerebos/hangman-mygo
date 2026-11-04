package app

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// A shake that stopped off center would move the row for good.
func TestTheShakeStartsAndEndsExactlyWhereTheRowLives(t *testing.T) {
	if shakeOffset(0) != 0 || abs(shakeOffset(1)) > 1e-5 {
		t.Errorf("offsets %v and %v", shakeOffset(0), shakeOffset(1))
	}
	if last := shakeFrames[len(shakeFrames)-1]; last.At != shakeDuration || abs(last.Value) > 1e-5 {
		t.Errorf("last frame %+v", last)
	}
}

func TestTheShakeStaysInsideItsDistanceAndActuallyShakes(t *testing.T) {
	var widest float32
	for i := range 1000 {
		widest = max(widest, abs(shakeOffset(float32(i)/1000)))
	}
	if widest > shakeDistance || widest < shakeDistance/2 {
		t.Errorf("widest %v", widest)
	}
}

// Every effect is one-shot: once it has played, the window asks for no more
// frames and the board looks exactly as it does at rest.
func TestEveryEffectSettles(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	tt.Key(0, wrongKey(a.game))
	tt.Key(0, ui.KeyA+ui.Key(a.game.Word()[0]-'A'))
	if !tt.FramePending() {
		t.Fatal("a guess played no animation")
	}
	tt.Advance(3 * time.Second)
	if tt.FramePending() {
		t.Error("the board kept asking for frames after its animations")
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// A replayed sequence starts from the value it last showed, so without the
// snap back to zero every replay after the first would sit at 1. The gallows
// drew its parts on only on the first wrong guess of a session that way.
func TestOncePlaysAgainFromTheStartEveryRun(t *testing.T) {
	run, value := 1, float32(0)
	view := func(c *ui.Context) {
		box := ui.Box(c.Key("box")).Size(10, 10)
		value = once(box, "t", run, 400*time.Millisecond, ui.Linear)
	}
	tt := ui.NewTester(view, 100, 100)
	tt.Advance(time.Second)
	if value != 1 {
		t.Fatalf("first run ended at %v", value)
	}
	run = 2
	tt.Frame()
	// On the very frame the run starts, not a moment later: a value that
	// starts from where it last was draws a new part whole for a frame.
	if value != 0 {
		t.Errorf("the second run starts at %v", value)
	}
	tt.Advance(200 * time.Millisecond)
	if value > 0.7 {
		t.Errorf("half way through the second run the value is %v", value)
	}
}

// The drawer springs open and shut rather than jumping. Checked part way,
// because a drawer that jumped looks the same at rest, which is how a guard
// that stopped its spring got past every other test.
func TestTheDrawerSpringsOpenAndShut(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	for _, open := range []bool{true, false} {
		// By clicking, as a player does: a click can build the view twice in
		// one frame, which is where the broken guard bit. Open from the menu,
		// shut with the drawer's own close button.
		if open {
			if err := tt.Click("More"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Lifetime stats"); err != nil {
				t.Fatal(err)
			}
		} else if err := tt.Click("Close"); err != nil {
			t.Fatal(err)
		}
		if a.showStats != open {
			t.Fatalf("the menu left the stats at %v", a.showStats)
		}
		tt.Advance(100 * time.Millisecond)
		if a.drawerShown <= 0.05 || a.drawerShown >= 0.95 {
			t.Errorf("open %v: drawer at %v part way", open, a.drawerShown)
		}
		tt.Advance(2 * time.Second)
		if a.drawerMoving {
			t.Errorf("open %v: drawer still moving at rest", open)
		}
	}
}

// With the desktop asking for less motion, nothing travels: the drawer is
// open at once, and a wrong guess (shake, draw-on, sway, pip, flight) is
// settled the moment it lands. Swaps that fade still fade in place, as mygo
// keeps fades under less motion, so they are not checked here.
func TestLessMotionMeansNothingTravels(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Lifetime stats"); err != nil {
		t.Fatal(err)
	}
	if a.drawerShown != 1 {
		t.Errorf("drawer at %v with less motion", a.drawerShown)
	}
	tt.Key(0, wrongKey(a.game))
	if tt.FramePending() {
		t.Error("a wrong guess still animates with less motion")
	}
}

// The first wrong guess of a later word draws its part on as the first word's
// did. The counts behind the effects start over with each word, and keyed by
// the count alone the gallows found its sequence already played and showed
// the part at once, with no sway, for every word after the first.
func TestASecondWordDrawsItsPartsOnToo(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	// Only the gallows' own box is compared, because other things move
	// after a wrong guess too.
	partWay := func() bool {
		tt.Key(0, wrongKey(a.game))
		box, ok := tt.Find(gallowsLabel(a.game.WrongGuesses(), a.game.GuessBudget()))
		if !ok {
			t.Fatal("no gallows on screen")
		}
		tt.Advance(60 * time.Millisecond)
		early := crop(tt.Image(), box)
		tt.Advance(3 * time.Second)
		return !sameImage(early, crop(tt.Image(), box))
	}
	if !partWay() {
		t.Fatal("the first word's wrong guess did not animate")
	}
	tt.Key(ui.Cmd, ui.KeyN)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(2 * time.Second)
	if !partWay() {
		t.Error("the second word's first wrong guess appeared at once")
	}
}

func sameImage(a, b *image.RGBA) bool {
	return a.Rect == b.Rect && bytes.Equal(a.Pix, b.Pix)
}

// difference is the mean difference of two images, per channel, from 0 to
// 255.
func difference(a, b *image.RGBA) float64 {
	if a.Rect != b.Rect {
		return 255
	}
	total := 0
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		total += max(d, -d)
	}
	return float64(total) / float64(len(a.Pix))
}

// crop copies the part of img under r, a box in DIPs at the tester's scale
// of 1.
func crop(img *image.RGBA, r ui.Rect) *image.RGBA {
	box := image.Rect(int(r.X), int(r.Y), int(r.X+r.W), int(r.Y+r.H))
	out := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
	for y := range box.Dy() {
		for x := range box.Dx() {
			out.Set(x, y, img.At(box.Min.X+x, box.Min.Y+y))
		}
	}
	return out
}

// When the winning letter is in several cells, all of its flights land with
// the word rather than late. A flight waiting to leave is held at its start,
// in the keyboard folding away, so staggered flights rode down toward the
// result card first and reached the word only after the win reveal was over.
func TestAWinningLetterInSeveralCellsLandsWithTheWord(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	g, err := game.FromWordsWithSeed(1, "BANANA")
	if err != nil {
		t.Fatal(err)
	}
	a.game = g
	a.freshWord(nil)
	tt.Frame()
	tt.Key(0, ui.KeyB)
	tt.Key(0, ui.KeyN)
	tt.Advance(2 * time.Second)
	tt.Key(0, ui.KeyA)
	// The row itself glides as the result card replaces the keyboard, so it
	// is found again at each moment and only what is inside it compared.
	row := func() *image.RGBA {
		box, ok := tt.Find(wordLabel("BANANA"))
		if !ok {
			t.Fatal("no word on screen")
		}
		return crop(tt.Image(), box)
	}
	tt.Advance(700 * time.Millisecond)
	early := row()
	tt.Advance(3 * time.Second)
	// Close rather than equal: the flown letters are still settling their
	// last fraction of a pixel. A letter missing from the row differs by far
	// more.
	if d := difference(early, row()); d > 2 {
		t.Errorf("the word was still filling in 700 ms after the winning guess (differs by %.1f)", d)
	}
}

// A letter in several cells, typed while the keyboard is still rising back
// for a new word, lands with the word too: the keyboard is moving then as
// well, and a delayed flight rode along with it.
func TestALetterTypedAsTheKeyboardReturnsLandsWithTheWord(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	g, err := game.FromWordsWithSeed(1, "ZZZ", "BANANA")
	if err != nil {
		t.Fatal(err)
	}
	a.game = g
	a.freshWord(nil)
	tt.Frame()
	for a.game.Word() != "BANANA" {
		tt.Key(ui.Cmd, ui.KeyN)
		tt.Advance(time.Second)
		tt.Key(0, ui.KeyEnter)
	}
	// At once, while the keyboard is on its way back.
	tt.Advance(30 * time.Millisecond)
	tt.Key(0, ui.KeyA)
	row := func() *image.RGBA {
		box, ok := tt.Find(wordLabel("BANANA"))
		if !ok {
			t.Fatal("no word on screen")
		}
		return crop(tt.Image(), box)
	}
	tt.Advance(700 * time.Millisecond)
	early := row()
	tt.Advance(3 * time.Second)
	if d := difference(early, row()); d > 2 {
		t.Errorf("the word was still filling in 700 ms after the guess (differs by %.1f)", d)
	}
}
