package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/settings"
)

// bananas is a window playing a list of two words, BANANA then CHERRY, each
// with a category and a clue, and what it has read out so far thrown away.
func bananas(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a, tt, _ := tester(t, settings.Settings{})
	a.loadWordList(`{"name": "Fruit", "words": [
		{"word": "Banana", "category": "Food", "clue": "Yellow."},
		{"word": "Cherry", "category": "Food", "clue": "Red."}]}`, nil)
	tt.Frame()
	tt.Announcements()
	if w := a.game.Word(); w != "BANANA" && w != "CHERRY" {
		t.Fatalf("dealt %q", a.game.Word())
	}
	return a, tt
}

// heardSaid is everything read out since the last call, as one string.
func heardSaid(tt *ui.Tester) string { return strings.Join(tt.Announcements(), " ") }

func TestEachGuessIsReadOut(t *testing.T) {
	a, tt := bananas(t)
	word := a.game.Word()
	in := rune(word[1]) // A in BANANA, H in CHERRY
	tt.Key(0, ui.KeyA+ui.Key(in-'A'))
	if got, want := heardSaid(tt), rightGuess(a.game, in); got != want {
		t.Errorf("a right letter: %q, want %q", got, want)
	}
	tt.Key(0, ui.KeyZ)
	if got, want := heardSaid(tt), "Z is not in the word. "+guessesLeft(a.game); got != want {
		t.Errorf("a wrong letter: %q, want %q", got, want)
	}
}

func TestTheWordsForAGuessCountItsLetters(t *testing.T) {
	g := mustWords(t, "BANANA")
	g.Guess('A')
	if got := rightGuess(g, 'A'); got != "A is in the word, 3 letters." {
		t.Errorf("%q", got)
	}
	if got := rightGuess(g, 'B'); got != "B is in the word, 1 letter." {
		t.Errorf("%q", got)
	}
	for wrong := 0; g.RemainingGuesses() > 1; wrong++ {
		g.Guess(rune('C' + wrong))
	}
	if got := guessesLeft(g); got != "1 guess left." {
		t.Errorf("%q", got)
	}
}

// Winning reads out the letter, the word and its points, in that order.
func TestWinningAWordIsReadOut(t *testing.T) {
	a, tt := bananas(t)
	winWord(a, tt)
	got := strings.Join(tt.Announcements(), " | ")
	won := wordEndSaid(a.game, a.session.MatchPoints(), false)
	if !strings.HasSuffix(got, won) || !strings.HasPrefix(won, "You won. The word was "+a.game.Word()+", for ") {
		t.Errorf("heard %q", got)
	}
}

func TestAHintAClueAndGivingUpAreReadOut(t *testing.T) {
	a, tt := bananas(t)
	tt.Key(ui.Cmd, ui.KeyL)
	if got := heardSaid(tt); got != "Clue: "+a.game.Clue() {
		t.Errorf("the clue: %q", got)
	}
	tt.Key(ui.Cmd, ui.KeyH)
	if got := heardSaid(tt); !strings.HasPrefix(got, "Hint: ") || !strings.HasSuffix(got, "It cost a guess. "+guessesLeft(a.game)) {
		t.Errorf("the hint: %q", got)
	}
	tt.Key(ui.Cmd, giveUpKey)
	if got, want := heardSaid(tt), "You gave up. The word was "+a.game.Word()+"."; got != want {
		t.Errorf("giving up: %q, want %q", got, want)
	}
}

// The next word says where it is in the match, and the summary how the
// match went.
func TestTheNextWordAndTheMatchAreReadOut(t *testing.T) {
	a, tt := bananas(t)
	winWord(a, tt)
	tt.Announcements()
	tt.Key(0, ui.KeyEnter)
	if got := heardSaid(tt); got != "Word 2 of 2, 6 letters, Food." {
		t.Errorf("the next word: %q", got)
	}
	winWord(a, tt)
	tt.Announcements()
	tt.Key(0, ui.KeyEnter)
	if got := heardSaid(tt); !strings.HasPrefix(got, "You won the match!") {
		t.Errorf("the summary: %q", got)
	}
}

// A new best streak is read out after the word that set it.
func TestANewBestStreakIsReadOut(t *testing.T) {
	a, tt, _ := onAStreak(2, 2)
	tt.Frame()
	tt.Announcements()
	winWord(a, tt)
	said := tt.Announcements()
	if len(said) == 0 || !strings.HasSuffix(said[len(said)-1], newBestSaid(3)) {
		t.Errorf("heard %q", said)
	}
}

// A letter typed at the Reset question guesses nothing, so it says nothing.
func TestNothingIsReadOutBehindTheResetQuestion(t *testing.T) {
	a, tt := bananas(t)
	a.confirmReset = true
	tt.Frame()
	tt.Announcements()
	tt.Key(0, ui.KeyA)
	if said := tt.Announcements(); slices.ContainsFunc(said, func(s string) bool { return strings.Contains(s, "in the word") }) {
		t.Errorf("heard %q", said)
	}
}
