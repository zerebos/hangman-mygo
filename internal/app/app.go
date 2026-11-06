// Package app is the Hangman window: its state, the actions that change it,
// and the view that draws it with mygo's native UI.
package app

import (
	"os"
	"time"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/sounds"
	"github.com/zerebos/hangman-mygo/internal/stats"
	"github.com/zerebos/hangman-mygo/internal/words"
)

// Speaker plays a cue's samples (mono, at sounds.Rate) after a delay. It must
// return at once, because the view calls it from a build.
type Speaker interface {
	Play(samples []float32, delay time.Duration)
}

// wordEndDelay is how long after a word's last letter its win or loss
// phrase comes in, so the letter's own sound is heard first.
const wordEndDelay = 160 * time.Millisecond

// Options is what the window needs from outside: where the settings go,
// how to play sounds, and how to ask the user for a file.
type Options struct {
	// Save keeps the settings for the next run: settings.json on a desktop,
	// the page's storage in a browser. Nil keeps nothing, which is what
	// tests use.
	Save func(settings.Settings)
	// Speaker plays the sound cues; nil is silent.
	Speaker Speaker
	// PickFile asks the user for a word list and calls done with its
	// contents on the main thread, or never when the user cancels. Nil
	// means there is no way to ask, as in a browser, and the window offers
	// no Open word list.
	PickFile func(done func(contents string, err error))
	// TouchFirst is whether the device's main pointer is a finger that
	// cannot hover, as a phone's or a tablet's is. The window then leaves
	// out what only a keyboard can use, until a key is pressed.
	TouchFirst bool
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
	// showStats is whether the lifetime stats drawer is open.
	showStats bool
	// window is the window the header draws its controls for, or nil.
	window Window
	// drawerShown is how far open the stats drawer is drawn in this frame,
	// from 0 to 1. View sets it from a spring before building the stage,
	// which sizes its keys from it.
	drawerShown float32
	// drawerMoving is whether the drawer is on its way open or shut in this
	// frame, while the stage follows it (see glideOn).
	drawerMoving bool
	// points are the points of the word just won, on their way into the
	// Score chip, and landings counts the times they arrived, so the chip
	// pops once for each.
	points   pointsFlight
	landings int
	// pointsNow is where those points are in this frame. View sets it
	// before building anything, because both the result card and the
	// header read it.
	pointsNow pointsPhase
	// best is a record streak waiting for its moment in the Best chip, and
	// bestPops counts the moments, so the chip pops once for each.
	best     bestMoment
	bestPops int
	// bestHeld is whether the Best chip still shows the beaten record in
	// this frame. View sets it before building anything.
	bestHeld bool
	// deal counts the words dealt, so the word on the board has a fresh
	// identity each time and the old one slides away as the new one comes.
	deal int
	// summaryShown is whether the finished match's summary screen is up,
	// in place of the stage. It stays down until the player asks for it, so
	// the last word's result is seen first.
	summaryShown bool
	// lastPack is the last word list loaded from a file, kept in memory only,
	// so Play again can deal its match afresh. A relaunch forgets it.
	lastPack *words.Pack
	// confetti is the burst a won match's summary throws.
	confetti []piece
	// gaveUpFrom is how many wrong guesses the word had when it was given
	// up on, so the gallows can draw the parts that giving up spent one
	// after another, or -1 when the word on the board was not given up on.
	gaveUpFrom int
	// fade is where the switch between light and dark is (see theme).
	fade themeState
	// confirmReset is whether the Reset stats question is open.
	confirmReset bool
	// keysPressed is whether a key has played the game since launch, which
	// on a touch-first device means a keyboard is attached after all.
	keysPressed bool
	// toast is a message for the view to show as a toast on its next build,
	// from work that finished outside it.
	toast string
}

// New builds the window's state from the settings the last run left behind,
// picking the match up where it was if it still describes a playable one.
func New(s settings.Settings, opts Options) *App {
	g, session := resumeOrStart(&s)
	// The mirror now says what is actually on the board, so a saved match
	// that was refused is not written back out by the next theme toggle.
	s.InFlight = saveMatch(g, session)
	return &App{opts: opts, settings: s, game: g, session: session, gaveUpFrom: -1}
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
	if a.opts.Save != nil {
		a.opts.Save(a.settings)
	}
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
	bestBefore := a.session.Stats().BestStreak
	earned := a.record(out.Word, out.Match)

	switch out.Result {
	case game.Correct:
		a.play(sounds.Wood.Right(a.rightStep()), 0)
	case game.Wrong:
		a.play(sounds.Wood.Wrong(a.wrongStep()), 0)
	}
	a.playWordEnd(out.Word)
	if best := a.session.Stats().BestStreak; isNewBest(bestBefore, best) {
		a.best = bestMoment{from: bestBefore, to: best}
		a.play(sounds.Wood.NewBest(), newBestDelay)
	}

	switch {
	case out.Result == game.Invalid:
		a.notice = bad(invalidGuess)
	case out.Word == game.Won:
		a.notice = good(gameWon)
		a.points = pointsFlight{points: earned, total: a.session.MatchPoints(), deal: a.deal}
	case out.Word == game.Lost:
		a.notice = bad(gameLost)
	default:
		// The original cleared the line on any ordinary guess.
		a.notice = nil
	}
	if landed {
		a.persist()
	}
}

// hint buys a letter with a wrong guess. To the rest of the view a hint is a
// guess, so the letter flies in and the gallows draws a part as for one. A
// hint never ends a word: the rules refuse the last guess and the last
// letter.
func (a *App) hint() {
	out := a.game.Hint()
	if out.Result != game.Revealed {
		return
	}
	a.lastGuess = out.Letter
	a.play(sounds.Wood.Hint(a.rightStep(), a.wrongStep()), 0)
	a.notice = good(hintGiven + " " + string(out.Letter) + ".")
	a.persist()
}

// play plays a cue, unless the sound is off or there is nothing to play it.
func (a *App) play(samples []float32, delay time.Duration) {
	if a.opts.Speaker != nil && !a.settings.Mute {
		a.opts.Speaker.Play(samples, delay)
	}
}

// playWordEnd plays a word's win or loss phrase just after the sound of the
// letter that ended it.
func (a *App) playWordEnd(word game.Result) {
	switch word {
	case game.Won:
		a.play(sounds.Wood.WordWon(), wordEndDelay)
	case game.Lost:
		a.play(sounds.Wood.WordLost(), wordEndDelay)
	}
}

// wrongStep is the note the wrong guess just spent plays, a hint's
// included: the semitones it has fallen to (sounds.WrongStep).
func (a *App) wrongStep() int {
	return sounds.WrongStep(a.game.WrongGuesses()-1, a.game.GuessBudget())
}

// rightStep is the note a right letter just guessed plays: how many of the
// word's distinct letters had been found before it, out of all of them, so
// the word climbs a note a letter to its octave on the last (sounds.RightStep).
func (a *App) rightStep() int {
	var distinct, found game.Letters
	for _, r := range a.game.Word() {
		distinct = distinct.With(r)
		if a.game.Guessed().Has(r) {
			found = found.With(r)
		}
	}
	return sounds.RightStep(found.Len()-1, distinct.Len())
}

// showClue puts the clue on screen. Nothing is recorded or charged.
func (a *App) showClue() {
	if canShowClue(a.game, a.clueShown) {
		a.clueShown = true
		a.play(sounds.Wood.Clue(), 0)
	}
}

// giveUp loses the word, and sounds like it: it ends the streak and scores
// nothing, as losing does. (The original game gave up in silence.) The guard
// keeps a finished word from being scored twice, because GiveUp refuses
// silently.
func (a *App) giveUp() {
	if a.game.IsWordOver() {
		return
	}
	a.gaveUpFrom = a.game.WrongGuesses()
	match := a.game.GiveUp()
	a.record(game.Lost, match)
	a.notice = bad(gaveUp)
	a.play(sounds.Wood.WordLost(), 0)
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

// showSummary puts the finished match's summary on screen.
func (a *App) showSummary() {
	if !a.game.IsMatchOver() || a.summaryShown {
		return
	}
	a.summaryShown = true
	if a.game.MatchOutcome() == game.MatchWon {
		a.confetti = newConfetti()
		a.play(sounds.Wood.MatchWon(), 0)
	} else {
		a.play(sounds.Wood.MatchEnded(), 0)
	}
}

// canReplay reports whether Play again has something to deal: the
// difficulty just finished, or the word list loaded this session.
func (a *App) canReplay() bool {
	return a.game.Difficulty() != game.NoDifficulty || a.lastPack != nil
}

// playAgain starts a new match on what the one just finished was played
// from.
func (a *App) playAgain() {
	if !a.game.IsMatchOver() {
		return
	}
	if d := a.game.Difficulty(); d != game.NoDifficulty {
		a.setDifficulty(d)
		return
	}
	if a.lastPack != nil && a.game.SetPack(*a.lastPack) == nil {
		a.session.StartMatch()
		a.freshWord(nil)
		a.persist()
	}
}

// freshWord clears what belongs to the word that is gone.
//
// Every way a fresh word is dealt comes through here (the next word, a new
// difficulty, a list loaded, Play again), so this is where the fresh word's
// cue plays.
func (a *App) freshWord(n *notice) {
	a.play(sounds.Wood.NewWord(), 0)
	a.gaveUpFrom = -1
	a.deal++
	a.summaryShown = false
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

// canOpenFiles is whether the window can ask for a word list.
func (a *App) canOpenFiles() bool { return a.opts.PickFile != nil }

// keysTaught is whether the window shows the keyboard's shortcuts and
// mentions its keys. A touch-first device has no keyboard until one of its
// keys is pressed, and a list of shortcuts it cannot press is only noise.
func (a *App) keysTaught() bool { return !a.opts.TouchFirst || a.keysPressed }

// openWordList asks for a file and loads it.
func (a *App) openWordList() {
	if !a.canOpenFiles() {
		return
	}
	a.opts.PickFile(a.loadWordList)
}

// loadWordList plays the list a file held, or says it could not.
func (a *App) loadWordList(contents string, err error) {
	charge, pack, ok := loadWords(a.game, contents, err)
	if !ok {
		a.notice = bad(fileError)
		return
	}
	a.lastPack = pack
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

func (a *App) toggleSound() {
	a.settings.Mute = !a.settings.Mute
	a.save()
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
