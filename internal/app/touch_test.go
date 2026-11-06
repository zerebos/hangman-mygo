package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/settings"
)

// touchTester is a window on a touch-first device.
func touchTester(touch bool) (*App, *ui.Tester) {
	a := New(settings.Settings{}, Options{TouchFirst: touch})
	return a, ui.NewTester(a.View, 1000, 800)
}

// keysMentioned reports whether the view teaches the keyboard: the
// shortcut bar, and the lines that name keys.
func keysMentioned(tt *ui.Tester) (bar, idle, enter bool) {
	return shows(tt, chord("H")), shows(tt, idleHint), shows(tt, "Press Enter for the next word.")
}

// A touch-first device shows no shortcuts and names no keys, and asks for
// a tap rather than typing.
func TestATouchFirstDeviceIsNotTaughtTheKeyboard(t *testing.T) {
	a, tt := touchTester(true)
	tt.Frame()
	if bar, idle, _ := keysMentioned(tt); bar || idle || !shows(tt, idleHintTouch) {
		t.Errorf("bar %v, typing hint %v, texts %q", bar, idle, tt.Texts())
	}
	// Taps guess through the keys' buttons, which press no keys.
	for _, r := range a.game.Word() {
		if 'A' <= r && r <= 'Z' {
			a.guess(r)
		}
	}
	tt.Frame()
	if !a.game.IsWordOver() {
		t.Fatal("the word is not over")
	}
	if bar, _, enter := keysMentioned(tt); bar || enter {
		t.Errorf("after the word: bar %v, Enter line %v", bar, enter)
	}
}

// A key pressed on a touch-first device means a keyboard is there after
// all, so the window teaches it from then on.
func TestAKeyPressBringsTheKeyboardBack(t *testing.T) {
	_, tt := touchTester(true)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyL)
	if bar, _, _ := keysMentioned(tt); !bar {
		t.Errorf("after a key: texts %q", tt.Texts())
	}
}

// Anywhere else the keyboard is taught from the start.
func TestADesktopIsTaughtTheKeyboard(t *testing.T) {
	_, tt := touchTester(false)
	tt.Frame()
	if bar, idle, _ := keysMentioned(tt); !bar || !idle {
		t.Errorf("bar %v, typing hint %v", bar, idle)
	}
}
