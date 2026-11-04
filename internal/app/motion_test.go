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
	tt.Advance(2 * time.Second)
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
