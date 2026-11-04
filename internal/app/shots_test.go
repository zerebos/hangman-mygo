package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// TestShots renders the window in a few states to PNGs, for looking at by
// eye. It runs only when HANGMAN_SHOTS names a directory to write them to:
//
//	HANGMAN_SHOTS=/tmp/shots go test ./internal/app -run TestShots
func TestShots(t *testing.T) {
	dir := os.Getenv("HANGMAN_SHOTS")
	if dir == "" {
		t.Skip("set HANGMAN_SHOTS to a directory to write screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	shot := func(tt *ui.Tester, name string) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}

	for _, theme := range []settings.Theme{settings.Dark, settings.Light} {
		suffix := "-dark"
		if theme == settings.Light {
			suffix = "-light"
		}
		a := New(settings.Settings{Theme: theme}, Options{})
		a.game = game.WithSeed(game.Easy, 7)
		a.SetWindow(&fakeWindow{})
		tt := ui.NewTester(a.View, 1000, 800)
		shot(tt, "start"+suffix)

		// The smallest window, with the drawer open and the dropdown down.
		a.showStats = true
		tt.SetSize(880, 660)
		tt.Frame()
		tt.Advance(time.Second)
		if err := tt.Click(subtitle(a.game)); err != nil {
			t.Fatal(err)
		}
		shot(tt, "min-menu"+suffix)
		tt.Key(0, ui.KeyEscape)
		a.showStats = false
		tt.SetSize(1000, 800)
		tt.Frame()
		tt.Advance(time.Second)

		// A couple of letters in, one right and one wrong.
		word := a.game.Word()
		a.guess(rune(word[0]))
		for r := 'Z'; r >= 'A'; r-- {
			if keyStateOf(a.game, r) == keyAvailable && !containsRune(word, r) {
				a.guess(r)
				break
			}
		}
		a.showClue()
		a.showStats = true
		tt.Frame()
		tt.Advance(time.Second)
		shot(tt, "playing"+suffix)

		a.showStats = false
		a.giveUp()
		tt.Frame()
		tt.Advance(90 * time.Millisecond)
		shot(tt, "mid-fold"+suffix)
		tt.Advance(time.Second)
		shot(tt, "lost"+suffix)

		// Part way through the drawer opening, the keys shrinking with it.
		a.showStats = true
		tt.Frame()
		tt.Advance(120 * time.Millisecond)
		shot(tt, "mid-drawer"+suffix)
		a.showStats = false
		tt.Frame()
		tt.Advance(time.Second)

		// Part way through a wrong guess: the shake, the pip's halo, the
		// key settling and the newest body part drawing on.
		a.nextWord()
		tt.Frame()
		tt.Key(0, ui.KeyA+ui.Key(wrongLetter(a.game)-'A'))
		tt.Advance(120 * time.Millisecond)
		shot(tt, "mid-wrong"+suffix)
		tt.Advance(time.Second)

		// Part way through a correct letter's flight into the word.
		tt.Key(0, ui.KeyA+ui.Key(rune(a.game.Word()[len(a.game.Word())-1])-'A'))
		tt.Advance(110 * time.Millisecond)
		shot(tt, "mid-right"+suffix)
		tt.Advance(time.Second)

		// Part way through the win reveal.
		for _, r := range a.game.Word() {
			if 'A' <= r && r <= 'Z' && !a.game.Guessed().Has(r) {
				a.guess(r)
			}
		}
		tt.Frame()
		tt.Advance(150 * time.Millisecond)
		shot(tt, "mid-win"+suffix)

		// A letter filling three cells, each flying from its own stand-in.
		banana, err := game.FromWordsWithSeed(1, "BANANA")
		if err != nil {
			t.Fatal(err)
		}
		a.game = banana
		a.freshWord(nil)
		tt.Frame()
		tt.Advance(time.Second)
		tt.Key(0, ui.KeyA)
		tt.Advance(150 * time.Millisecond)
		shot(tt, "mid-triple"+suffix)
		tt.Advance(time.Second)

		// A won match's summary: counting up under the confetti, then at rest.
		short, err := game.FromWordsWithSeed(3, "AB", "CD", "EF")
		if err != nil {
			t.Fatal(err)
		}
		a.game = short
		tt.Frame()
		for !a.game.IsMatchOver() {
			for _, r := range a.game.Word() {
				a.guess(r)
			}
			if a.game.WordNumber() == 2 {
				a.giveUp()
			}
			a.nextWord()
		}
		a.showSummary()
		tt.Frame()
		tt.Advance(600 * time.Millisecond)
		shot(tt, "summary-mid"+suffix)
		tt.Advance(4 * time.Second)
		shot(tt, "summary"+suffix)
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// TestGallowsFrames renders a wrong guess at several moments, to check the
// draw-on and the sway by eye. Gated on HANGMAN_SHOTS like TestShots.
func TestGallowsFrames(t *testing.T) {
	dir := os.Getenv("HANGMAN_SHOTS")
	if dir == "" {
		t.Skip("set HANGMAN_SHOTS to a directory to write screenshots")
	}
	a := New(settings.Settings{}, Options{})
	a.game = game.WithSeed(game.Insane, 7)
	tt := ui.NewTester(a.View, 1000, 800)
	for i := range 3 {
		tt.Key(0, ui.KeyA+ui.Key(wrongLetter(a.game)-'A'))
		if i < 2 {
			tt.Advance(3 * time.Second)
		}
	}
	elapsed := time.Duration(0)
	for _, at := range []time.Duration{60, 180, 320, 500, 800, 1200} {
		tt.Advance(at*time.Millisecond - elapsed)
		elapsed = at * time.Millisecond
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("gallows-%04d.png", at)))
		if err != nil {
			t.Fatal(err)
		}
		_ = png.Encode(f, tt.Image())
		f.Close()
	}
}

// TestSketchFrames renders the clean and the hand-drawn gallows side by side,
// part way through a part and with a finished figure. Gated like TestShots.
func TestSketchFrames(t *testing.T) {
	dir := os.Getenv("HANGMAN_SHOTS")
	if dir == "" {
		t.Skip("set HANGMAN_SHOTS to a directory to write screenshots")
	}
	for _, theme := range []settings.Theme{settings.Dark, settings.Light} {
		for _, sketch := range []bool{false, true} {
			a := New(settings.Settings{Theme: theme, Sketch: sketch}, Options{})
			a.game = game.WithSeed(game.Insane, 7)
			tt := ui.NewTester(a.View, 1000, 800)
			for range 4 {
				tt.Key(0, ui.KeyA+ui.Key(wrongLetter(a.game)-'A'))
				tt.Advance(3 * time.Second)
			}
			tt.Key(0, ui.KeyA+ui.Key(wrongLetter(a.game)-'A'))
			tt.Advance(250 * time.Millisecond)
			name := fmt.Sprintf("sketch-%v-%v", sketch, theme == settings.Light)
			save := func(suffix string) {
				f, err := os.Create(filepath.Join(dir, name+suffix+".png"))
				if err != nil {
					t.Fatal(err)
				}
				_ = png.Encode(f, tt.Image())
				f.Close()
			}
			save("-mid")
			tt.Advance(3 * time.Second)
			tt.Key(0, ui.KeyA+ui.Key(wrongLetter(a.game)-'A'))
			tt.Advance(4 * time.Second)
			save("-full")
		}
	}
}
