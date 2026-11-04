// Package app is the Hangman window: its state, the actions that change it,
// and the view that draws it with mygo's native UI.
package app

import (
	"os"
	"strconv"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/stats"
)

// Cues plays the original's two sound cues. Giving up is silent, as it was
// in the original.
type Cues interface {
	PlayWin()
	PlayLoss()
}

type silent struct{}

func (silent) PlayWin()  {}
func (silent) PlayLoss() {}

// Options is what the window needs from outside: where the settings live,
// how to play sounds, and how to ask the user for a file.
type Options struct {
	// SettingsPath is where settings.json is written. Empty writes nothing,
	// which is what tests use.
	SettingsPath string
	// Cues plays the win and loss sounds; nil is silent.
	Cues Cues
	// PickFile asks the user for a word list and calls done with its
	// contents on the main thread, or never when the user cancels. Nil
	// leaves Open word list doing nothing.
	PickFile func(done func(contents string, err error))
}

// App is the state of the Hangman window.
type App struct {
	opts     Options
	settings settings.Settings
	game     *game.Game
	session  *stats.Session

	// notice is the latest per-word message, or nil for the idle line. The
	// end-of-match line is derived instead, by matchSummary.
	notice *notice
	// lastGuess is the letter of the last guess that landed, correct or
	// wrong. It tells the word which cells to reveal and the keyboard which
	// key to settle, and it is cleared whenever a fresh word starts.
	lastGuess rune
	// clueShown is whether the word's clue is on screen. It is the view's
	// own, because reading a clue changes nothing that is scored.
	clueShown bool
	// showStats is whether the lifetime stats fold out under the board.
	showStats bool
	// drawerShown is how far open the stats drawer is drawn in this frame,
	// from 0 to 1. View sets it from a spring before building the stage,
	// which sizes its keys from it.
	drawerShown float32
	// deal counts the words dealt, so the word on the board has a fresh
	// identity each time and the old one slides away as the new one comes.
	deal int
	// confirmReset is whether the Reset stats question is open.
	confirmReset bool
	// toast is a message for the view to show as a toast on its next build,
	// from work that finished outside it.
	toast string
}

// New builds the window's state from the settings the last run left behind,
// picking the match up where it was if it still describes a playable one.
func New(s settings.Settings, opts Options) *App {
	if opts.Cues == nil {
		opts.Cues = silent{}
	}
	g, session := resumeOrStart(&s)
	// The mirror now says what is actually on the board, so a saved match
	// that was refused is not written back out by the next theme toggle.
	s.InFlight = saveMatch(g, session)
	return &App{opts: opts, settings: s, game: g, session: session}
}

// persist writes everything derived from the game and the session back to
// disk. Every action that changed either ends with this, on every guess
// rather than at exit, so killing the process is not a free reroll.
func (a *App) persist() {
	a.settings.Stats = *a.session.Stats()
	a.settings.InFlight = saveMatch(a.game, a.session)
	a.save()
}

func (a *App) save() {
	a.settings.Save(a.opts.SettingsPath)
}

// guess guesses a letter, typed or clicked.
func (a *App) guess(letter rune) {
	out := a.game.Guess(letter)
	if out.Result == game.Ignored {
		return
	}
	// Only a guess that changed something is worth animating, or saving.
	landed := out.Result == game.Correct || out.Result == game.Wrong
	if landed {
		a.lastGuess = upper(letter)
	}
	earned := a.record(out.Word, out.Match)

	switch {
	case out.Result == game.Invalid:
		a.notice = bad(invalidGuess)
	case out.Word == game.Won:
		a.opts.Cues.PlayWin()
		a.notice = good(gameWon + " +" + strconv.FormatUint(uint64(earned), 10))
	case out.Word == game.Lost:
		a.opts.Cues.PlayLoss()
		a.notice = bad(gameLost)
	default:
		// The original cleared the line on any ordinary guess.
		a.notice = nil
	}
	if landed {
		a.persist()
	}
}

// hint buys a letter with a wrong guess. Everything after the call is the
// guess path, because to the rest of the view a hint is a guess.
func (a *App) hint() {
	out := a.game.Hint()
	if out.Result != game.Revealed {
		return
	}
	a.lastGuess = out.Letter
	earned := a.record(out.Word, out.Match)
	switch out.Word {
	case game.Won:
		a.opts.Cues.PlayWin()
		a.notice = good(gameWon + " +" + strconv.FormatUint(uint64(earned), 10))
	case game.Lost:
		// Unreachable, because a hint refuses at one guess left, but the
		// rules are the game's to keep.
		a.opts.Cues.PlayLoss()
		a.notice = bad(gameLost)
	default:
		a.notice = good(hintGiven + " " + string(out.Letter) + ".")
	}
	a.persist()
}

// showClue puts the clue on screen. Nothing is recorded or charged.
func (a *App) showClue() {
	if canShowClue(a.game, a.clueShown) {
		a.clueShown = true
	}
}

// giveUp loses the word. The guard keeps a finished word from being scored
// twice, because GiveUp refuses silently.
func (a *App) giveUp() {
	if a.game.IsWordOver() {
		return
	}
	match := a.game.GiveUp()
	a.record(game.Lost, match)
	a.notice = bad(gaveUp)
	a.persist()
}

// record puts a finished word, and the match when it was the last, on the
// scoreboard, and returns what the word scored.
func (a *App) record(word game.Result, match game.MatchOutcome) uint32 {
	if word == game.InPlay {
		return 0
	}
	d := a.game.Difficulty()
	// Read after the guess landed: the budget still unspent is what the
	// word is worth.
	earned := a.session.RecordWord(d, word, a.game.RemainingGuesses())
	if match != game.MatchRunning {
		a.session.RecordMatch(d, match)
	}
	return earned
}

// nextWord deals the next word of the match. The match score carries on.
func (a *App) nextWord() {
	if a.game.NextWord() {
		a.freshWord(nil)
		a.persist()
	}
}

// freshWord clears what belongs to the word that is gone.
func (a *App) freshWord(n *notice) {
	a.deal++
	a.notice = n
	a.lastGuess = 0
	a.clueShown = false
}

// recordAbandoned charges a word walked out on as a loss and says so.
func (a *App) recordAbandoned(charge *abandonCharge) *notice {
	if charge == nil {
		return nil
	}
	a.session.RecordWord(charge.difficulty, game.Lost, 0)
	return bad("Leaving " + charge.word + " counts as a loss in my book.")
}

// setDifficulty switches to d. A click on the difficulty in play mid-match
// changes nothing; once the match is over it restarts it.
func (a *App) setDifficulty(d game.Difficulty) {
	charge, ok := switchDifficulty(a.game, d)
	if !ok {
		return
	}
	// A fresh match, so the match score starts again. The streak is left
	// alone on purpose.
	a.session.StartMatch()
	a.freshWord(a.recordAbandoned(charge))
	a.settings.Difficulty = settings.Difficulty(d)
	a.persist()
}

// openWordList asks for a file and loads it.
func (a *App) openWordList() {
	if a.opts.PickFile == nil {
		return
	}
	a.opts.PickFile(a.loadWordList)
}

// loadWordList plays the list a file held, or says it could not.
func (a *App) loadWordList(contents string, err error) {
	charge, ok := loadWords(a.game, contents, err)
	if !ok {
		a.notice = bad(fileError)
		return
	}
	a.session.StartMatch()
	a.freshWord(a.recordAbandoned(charge))
	a.toast = loadedSummary(a.game.PackWords(), a.game.TotalWords())
	a.persist()
}

// resetStats throws the lifetime tally away, once the question is answered.
func (a *App) resetStats() {
	a.session.ResetStats()
	a.persist()
}

func (a *App) toggleTheme() {
	if a.settings.Theme == settings.Dark {
		a.settings.Theme = settings.Light
	} else {
		a.settings.Theme = settings.Dark
	}
	a.save()
}

// ReadFile reads a word list for PickFile.
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func upper(r rune) rune {
	if 'a' <= r && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}
