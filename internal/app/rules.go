package app

import (
	"fmt"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/stats"
	"github.com/zerebos/hangman-mygo/internal/words"
)

// The helpers in this file only read the game, or change it in a fixed
// order, and take no UI types, so tests call them directly.

// The original's alert strings, verbatim.
const (
	gameLost     = "Bring Add/Drop Form!"
	gameWon      = "You WIN!"
	gaveUp       = "Giving up counts as a loss in my book."
	invalidGuess = "You managed to guess an invalid character, congrats."
	fileError    = "Sorry, we couldnt read in your file."
)

const (
	idleHint           = "Type a letter, or click one above."
	customListSubtitle = "Custom word list"
	hintGiven          = "That cost you a guess:"

	hintTooltip           = "Reveal a letter — costs one wrong guess"
	hintTooltipLastGuess  = "No hint: it would cost the last guess you have"
	hintTooltipLastLetter = "No hint: the last letter is yours to find"
	hintTooltipOver       = "No hint: this word is already finished"
	clueTooltip           = "Show what the word means — costs nothing"
	clueTooltipNone       = "No clue: this word list does not carry one"
	clueTooltipShown      = "The clue is already on screen"

	resetTitle   = "Reset lifetime stats?"
	resetOK      = "Reset"
	resetCancel  = "Keep them"
	resetNothing = "There is nothing saved yet, so this costs you nothing."
	// resetForever names the match score too, because ResetStats clears it.
	// The word itself survives, which is what a player is most likely to
	// worry about.
	resetForever = "go back to zero in every difficulty, along with the score of the match you " +
		"are playing. There is no undo, though the word itself is not touched."
)

// notice is a line of feedback in the style of the original's alertMessage:
// italic, green for good news and red for bad.
type notice struct {
	text string
	good bool
}

func good(text string) *notice { return &notice{text: text, good: true} }
func bad(text string) *notice  { return &notice{text: text} }

// keyState is how one letter key looks.
type keyState int

const (
	keyAvailable keyState = iota
	keyCorrect
	keyWrong
	keyOutOfPlay
)

func keyStateOf(g *game.Game, letter rune) keyState {
	if !g.Guessed().Has(letter) {
		if g.IsWordOver() {
			return keyOutOfPlay
		}
		return keyAvailable
	}
	for _, r := range g.Word() {
		if r == letter {
			return keyCorrect
		}
	}
	return keyWrong
}

// guessCount is how many letters have landed on this word, hints included.
// The animations key on it so each guess plays them again.
func guessCount(g *game.Game) int { return g.Guessed().Len() }

// subtitle is where the word came from: the difficulty, the pack's own name,
// or our wording for a pack that gave none. A pack reached through a
// difficulty is always called by the difficulty, so a downloaded pack cannot
// relabel the ladder.
func subtitle(g *game.Game) string {
	if d := g.Difficulty(); d != game.NoDifficulty {
		return d.Label()
	}
	if name := g.PackName(); name != "" {
		return name
	}
	return customListSubtitle
}

// canShowClue reports whether showing the clue would put anything new on
// screen. Unlike a hint it has nothing to do with the budget, so it stays
// available on the last guess and after the word is over.
func canShowClue(g *game.Game, clueShown bool) bool {
	return g.Clue() != "" && !clueShown
}

func clueTooltipFor(g *game.Game, clueShown bool) string {
	switch {
	case g.Clue() == "":
		return clueTooltipNone
	case clueShown:
		return clueTooltipShown
	}
	return clueTooltip
}

// hintTooltipFor says what the Hint button will do, or why it will not. The
// reason is why the button stays on screen greyed out instead of vanishing.
func hintTooltipFor(g *game.Game) string {
	switch g.HintRefusal() {
	case game.Revealed:
		return hintTooltip
	case game.NoGuessToSpare:
		return hintTooltipLastGuess
	case game.LastLetter:
		return hintTooltipLastLetter
	}
	return hintTooltipOver
}

// shortcut is one entry of the keyboard legend.
type shortcut int

const (
	shortcutHint shortcut = iota
	shortcutClue
	shortcutChangeWord
	shortcutOpenWordList
	shortcutNextWord
	shortcutSeeMatch
	shortcutPlayAgain
)

func (s shortcut) label() string {
	switch s {
	case shortcutHint:
		return "Reveal a letter"
	case shortcutClue:
		return "Show what the word means"
	case shortcutChangeWord:
		return "Give up on this word"
	case shortcutOpenWordList:
		return "Open a word list"
	case shortcutSeeMatch:
		return "See the match"
	case shortcutPlayAgain:
		return "Play again"
	}
	return "Next word"
}

// legendEntry is a legend entry and whether pressing it now would change
// anything. A key the game would refuse is dimmed rather than dropped, so the
// strip keeps its shape.
type legendEntry struct {
	shortcut shortcut
	live     bool
}

// shortcutLegend is what the legend lists for the game in hand. The four
// standing shortcuts are always there. Enter is listed only while it does
// something, under the name of what it does: the next word, the match's
// summary once the last word is over, and a new match from the summary when
// there is something to replay.
func shortcutLegend(g *game.Game, clueShown, summaryShown, canReplay bool) []legendEntry {
	hints := []legendEntry{
		{shortcutHint, g.CanHint()},
		{shortcutClue, canShowClue(g, clueShown)},
		{shortcutChangeWord, !g.IsWordOver()},
		{shortcutOpenWordList, true},
	}
	switch {
	case g.IsWordOver() && !g.IsMatchOver():
		hints = append(hints, legendEntry{shortcutNextWord, true})
	case g.IsMatchOver() && !summaryShown:
		hints = append(hints, legendEntry{shortcutSeeMatch, true})
	case g.IsMatchOver() && canReplay:
		hints = append(hints, legendEntry{shortcutPlayAgain, true})
	}
	return hints
}

// plural is "3 words" and "1 word".
func plural(count uint64, noun string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}

func points(n uint32) string { return plural(uint64(n), "point") }

// percent is a whole-number percentage of a rate from 0 to 1, as "75%".
func percent(rate float32) string { return fmt.Sprintf("%.0f%%", rate*100) }

// resetStatsSummary is what the Reset stats dialog says is about to go, in
// the terms the stats panel behind it shows.
func resetStatsSummary(s *stats.Stats) string {
	// IsEmpty rather than the three numbers below, because a hand-edited file
	// can hold a match count with none of them behind it.
	if s.IsEmpty() {
		return resetNothing
	}
	return fmt.Sprintf("%s, %s and a best streak of %d %s",
		plural(s.Points, "point"), plural(uint64(s.WordsPlayed()), "word"), s.BestStreak, resetForever)
}

// matchSummary is the end-of-match line, or nil while the match runs. It is
// derived on every frame so it cannot drift from the score it quotes.
func matchSummary(g *game.Game, s *stats.Session) *notice {
	wins := g.WordsWon()
	total := wins + g.WordsLost()
	scored := points(s.MatchPoints())
	switch g.MatchOutcome() {
	case game.MatchWon:
		return good(fmt.Sprintf("Good job, you got %d out of %d for %s", wins, total, scored))
	case game.MatchLost:
		return bad(fmt.Sprintf("Nice try, you only got %d out of %d for %s", wins, total, scored))
	case game.MatchTied:
		return good(fmt.Sprintf("A tie, not bad, you got %d out of %d for %s", wins, total, scored))
	}
	return nil
}

// loadedSummary is the toast after a list loads. A pack bigger than a match
// is normal, and "loaded 200 words" alone would be a half-truth.
func loadedSummary(packWords, matchWords int) string {
	if packWords > matchWords {
		return fmt.Sprintf("Loaded %d words — playing %d of them.", packWords, matchWords)
	}
	return fmt.Sprintf("Loaded %s. New match!", plural(uint64(packWords), "word"))
}

// abandonCharge is what walking out on a word costs: the word, and the
// difficulty it was dealt from. The difficulty is why this is a type: a reset
// changes it out from under the loss.
type abandonCharge struct {
	word       string
	difficulty game.Difficulty
}

// wordBeingAbandoned is the charge for the word on the board, or nil when
// walking away from it is free. It has to be read before the reset.
func wordBeingAbandoned(g *game.Game) *abandonCharge {
	if !g.HasWordToLose() {
		return nil
	}
	return &abandonCharge{word: g.Word(), difficulty: g.Difficulty()}
}

// switchDifficulty switches to d and says what the word it replaced cost, or
// reports false when the switch was refused.
//
// The order is the rule: the charge is read before SetDifficulty deals the
// new pool, because afterwards the game reports the difficulty switched to,
// and the loss belongs to the one the word came from. Whether the switch
// happened at all is SetDifficulty's own answer, so a charge is never handed
// back for a switch that did not happen.
func switchDifficulty(g *game.Game, d game.Difficulty) (*abandonCharge, bool) {
	charge := wordBeingAbandoned(g)
	if !g.SetDifficulty(d) {
		return nil, false
	}
	return charge, true
}

// loadWords deals a fresh pool from the contents of a file the player picked,
// and says what the word it replaced cost, or reports false when the file
// could not be used.
//
// The charge is read before the load but handed back only on success: a file
// that will not open, will not parse or holds nothing playable leaves the word
// on the board, so charging for it would take a word the player still has.
//
// The pack is handed back too, so a finished match on it can be played again.
func loadWords(g *game.Game, contents string, readErr error) (*abandonCharge, *words.Pack, bool) {
	charge := wordBeingAbandoned(g)
	if readErr != nil {
		return nil, nil, false
	}
	pack, err := words.Parse(contents)
	if err != nil {
		return nil, nil, false
	}
	if g.SetPack(pack) != nil {
		return nil, nil, false
	}
	return charge, &pack, true
}

// resumeOrStart is the match to open the window on: the one the last run left
// if it is still playable, and a fresh one otherwise. The game and the
// session come back together, because the match score lives in the session.
func resumeOrStart(s *settings.Settings) (*game.Game, *stats.Session) {
	if s.InFlight != nil {
		if g := game.Resume(s.InFlight.Snapshot()); g != nil {
			return g, stats.ResumeSession(s.Stats, s.InFlight.MatchPoints)
		}
	}
	d := game.Difficulty(s.Difficulty)
	if d == game.NoDifficulty {
		d = game.Easy
	}
	return game.New(d), stats.NewSession(s.Stats)
}

// saveMatch is the match in flight as the settings file stores it, or nil
// when the match is over and there is nothing to come back to.
func saveMatch(g *game.Game, s *stats.Session) *settings.SavedMatch {
	snapshot, ok := g.Snapshot()
	if !ok {
		return nil
	}
	return settings.NewSavedMatch(snapshot, s.MatchPoints())
}
