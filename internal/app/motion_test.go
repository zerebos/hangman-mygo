package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

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
	tt.Advance(200 * time.Millisecond)
	if value > 0.7 {
		t.Errorf("half way through the second run the value is %v", value)
	}
}
