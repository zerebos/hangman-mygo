package app

import (
	"fmt"
	"strings"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// What the game reads out to screen readers, as it happens. The board says
// what it is through its labels (the word's length, each played key, the
// gallows), but a guess changes it without moving the focus, so nothing
// would read the change out without these.
//
// The words are plain, unlike the original's alerts on screen, because a
// joke heard without the picture is only confusing.

// say queues text for screen readers to read out once, on the next frame.
// Announcing needs a *ui.Context, which actions don't have, so View hands
// the queue on, as it does a toast.
func (a *App) say(parts ...string) {
	for _, p := range parts {
		if p != "" {
			a.announcement = append(a.announcement, p)
		}
	}
}

// guessesLeft is how many wrong guesses the word can still afford.
func guessesLeft(g *game.Game) string {
	switch n := g.RemainingGuesses(); n {
	case 0:
		return "No guesses left."
	case 1:
		return "1 guess left."
	default:
		return fmt.Sprintf("%d guesses left.", n)
	}
}

// letters is a count of cells, as "1 letter" or "3 letters".
func letters(n int) string {
	if n == 1 {
		return "1 letter"
	}
	return fmt.Sprintf("%d letters", n)
}

// rightGuess is a letter found in the word, and how many cells it filled.
func rightGuess(g *game.Game, letter rune) string {
	return fmt.Sprintf("%c is in the word, %s.", letter, letters(occurrences(g.Word(), letter)))
}

// wrongGuess is a letter not in the word, and what the word can still afford.
func wrongGuess(g *game.Game, letter rune) string {
	return fmt.Sprintf("%c is not in the word. %s", letter, guessesLeft(g))
}

// hintGivenSaid is a hint: the letter it gave, and the guess it cost.
func hintGivenSaid(g *game.Game, letter rune) string {
	return fmt.Sprintf("Hint: %c, %s. It cost a guess. %s", letter, letters(occurrences(g.Word(), letter)), guessesLeft(g))
}

// wordEndSaid is a finished word: won or lost, the word, and what it scored.
func wordEndSaid(g *game.Game, earned uint32, gaveUp bool) string {
	switch {
	case g.IsWon():
		return fmt.Sprintf("You won. The word was %s, for %d points.", g.Word(), earned)
	case gaveUp:
		return fmt.Sprintf("You gave up. The word was %s.", g.Word())
	}
	return fmt.Sprintf("You lost. The word was %s.", g.Word())
}

// dealtSaid is a word dealt: where it is in the match, how long it is, and
// its category.
func dealtSaid(g *game.Game) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Word %d of %d, %s", g.WordNumber(), g.TotalWords(), letters(len([]rune(g.Word()))))
	if category := g.Category(); category != "" {
		b.WriteString(", " + category)
	}
	b.WriteString(".")
	return b.String()
}

// newBestSaid is a streak beating the best.
func newBestSaid(best uint32) string { return fmt.Sprintf("New best streak: %d.", best) }

// wordSaid is the word as it stands, for the read-the-word key: its length,
// each cell in order with the hidden ones as "blank", the letters missed so
// far, and what the word can still afford. It tells no more than the screen
// shows, because a finished word has every cell revealed already.
func wordSaid(g *game.Game) string {
	cells := g.Cells()
	spoken := make([]string, len(cells))
	for i, cell := range cells {
		spoken[i] = cellSaid(cell)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s.", letters(len(cells)), strings.Join(spoken, ", "))
	var missed []string
	for r := 'A'; r <= 'Z'; r++ {
		if g.Guessed().Has(r) && !strings.ContainsRune(g.Word(), r) {
			missed = append(missed, string(r))
		}
	}
	if len(missed) == 0 {
		b.WriteString(" No misses.")
	} else {
		b.WriteString(" Missed: " + strings.Join(missed, ", ") + ".")
	}
	if !g.IsWordOver() {
		b.WriteString(" " + guessesLeft(g))
	}
	return b.String()
}

// cellSaid is one cell of the word. Punctuation is named, because screen
// readers often skip it at their default settings.
func cellSaid(cell game.Cell) string {
	switch {
	case !cell.Revealed:
		return "blank"
	case cell.Value == ' ':
		return "space"
	case cell.Value == '/':
		return "slash"
	case cell.Value == '-':
		return "dash"
	case cell.Value == '\'':
		return "apostrophe"
	}
	return string(cell.Value)
}
