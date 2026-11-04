package app

import (
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
		tt := ui.NewTester(a.View, 1000, 800)
		shot(tt, "start"+suffix)

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
		shot(tt, "playing"+suffix)

		a.giveUp()
		tt.Frame()
		shot(tt, "lost"+suffix)

		// Part way through a wrong guess: the shake, the pip's halo, the
		// key settling and the newest body part drawing on.
		a.nextWord()
		tt.Frame()
		tt.Key(0, ui.KeyA+ui.Key(wrongLetter(a.game)-'A'))
		tt.Advance(120 * time.Millisecond)
		shot(tt, "mid-wrong"+suffix)

		// Part way through the win reveal.
		for _, r := range a.game.Word() {
			if 'A' <= r && r <= 'Z' && !a.game.Guessed().Has(r) {
				a.guess(r)
			}
		}
		tt.Frame()
		tt.Advance(150 * time.Millisecond)
		shot(tt, "mid-win"+suffix)
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
