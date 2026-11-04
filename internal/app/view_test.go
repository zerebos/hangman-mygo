package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/stats"
)

// tester opens the view on a fresh app at the default window size, with the
// settings written to a temporary file.
func tester(t *testing.T, s settings.Settings) (*App, *ui.Tester, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	a := New(s, Options{SettingsPath: path})
	return a, ui.NewTester(a.View, 1000, 800), path
}

type recorder struct{ wins, losses int }

func (r *recorder) PlayWin()  { r.wins++ }
func (r *recorder) PlayLoss() { r.losses++ }

func TestClickingAKeyGuessesItsLetter(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	letter := rune(a.game.Word()[0])
	if err := tt.Click(string(letter)); err != nil {
		t.Fatal(err)
	}
	if !a.game.Guessed().Has(letter) {
		t.Errorf("guessed %q after clicking %c", a.game.Guessed(), letter)
	}
}

func TestTypingALetterGuessesIt(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	tt.Key(0, ui.KeyQ)
	tt.Key(ui.Shift, ui.KeyX)
	if !a.game.Guessed().Has('Q') || !a.game.Guessed().Has('X') {
		t.Errorf("guessed %q", a.game.Guessed())
	}
}

// Ctrl+H spends a guess on a letter; a bare H is a guess of H, not a hint.
func TestCtrlHBuysALetterAndABareHGuessesOne(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	tt.Key(ui.Cmd, ui.KeyH)
	if a.game.WrongGuesses() != 1 || guessCount(a.game) != 1 {
		t.Fatalf("after Ctrl+H: %d wrong, guessed %q", a.game.WrongGuesses(), a.game.Guessed())
	}
	hinted := a.game.Guessed()
	tt.Key(0, ui.KeyH)
	if !a.game.Guessed().Has('H') || (!hinted.Has('H') && guessCount(a.game) != 2) {
		t.Errorf("a bare H did not guess H: %q", a.game.Guessed())
	}
}

func TestEnterDealsTheNextWordOnlyOnceTheWordIsOver(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	tt.Key(0, ui.KeyEnter)
	if a.game.WordNumber() != 1 {
		t.Fatal("Enter dealt a word mid-game")
	}
	tt.Key(ui.Cmd, ui.KeyN)
	if !a.game.IsWordOver() || a.notice == nil || a.notice.text != gaveUp {
		t.Fatalf("Ctrl+N did not give up: %+v", a.notice)
	}
	tt.Key(0, ui.KeyEnter)
	if a.game.WordNumber() != 2 || a.game.IsWordOver() {
		t.Errorf("word %d after Enter", a.game.WordNumber())
	}
}

func TestTheResetQuestionKeepsTheBoardOutOfItsWay(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{Stats: stats.Stats{Points: 500, WordsWon: 3, BestStreak: 3}})
	a.showStats = true
	tt.Frame()
	if err := tt.Click("Reset stats"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(resetTitle) {
		t.Fatalf("no question on screen: %q", tt.Texts())
	}
	// A letter typed at the question is not guessed on the board behind it.
	tt.Key(0, ui.KeyE)
	tt.Key(ui.Cmd, ui.KeyH)
	if a.game.Guessed() != 0 {
		t.Errorf("the board took %q through the dialog", a.game.Guessed())
	}
	if err := tt.Click(resetCancel); err != nil {
		t.Fatal(err)
	}
	if a.session.Stats().Points != 500 {
		t.Fatal("Keep them threw the stats away")
	}
	if err := tt.Click("Reset stats"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(resetOK); err != nil {
		t.Fatal(err)
	}
	if !a.session.Stats().IsEmpty() {
		t.Errorf("stats %+v after Reset", *a.session.Stats())
	}
}

func TestPickingADifficultyStartsAMatchOnIt(t *testing.T) {
	a, tt, path := tester(t, settings.Settings{})
	if err := tt.Click("Insane"); err != nil {
		t.Fatal(err)
	}
	if a.game.Difficulty() != game.Insane || a.game.GuessBudget() != 6 {
		t.Fatalf("difficulty %v, budget %d", a.game.Difficulty(), a.game.GuessBudget())
	}
	saved := settings.Load(path)
	if game.Difficulty(saved.Difficulty) != game.Insane || saved.InFlight == nil {
		t.Errorf("saved %+v", saved)
	}
	// The pill already selected does nothing mid-match.
	word := a.game.Word()
	tt.Key(0, ui.KeyQ)
	if err := tt.Click("Insane"); err != nil {
		t.Fatal(err)
	}
	if a.game.Word() != word || !a.game.Guessed().Has('Q') {
		t.Error("re-clicking the difficulty in play threw the word away")
	}
}

func TestSwitchingMidWordChargesTheWordAndSaysSo(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	word := a.game.Word()
	tt.Key(0, wrongKey(a.game))
	if err := tt.Click("Hard"); err != nil {
		t.Fatal(err)
	}
	if a.session.Stats().WordsLost != 1 || a.session.Stats().ForDifficulty(game.Easy).WordsLost != 1 {
		t.Error("the abandoned word was not charged to Easy")
	}
	if !tt.HasText("Leaving " + word + " counts as a loss in my book.") {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestWinningPlaysTheWinCueAndScores(t *testing.T) {
	cues := &recorder{}
	a := New(settings.Settings{}, Options{Cues: cues})
	tt := ui.NewTester(a.View, 1000, 800)
	for _, r := range a.game.Word() {
		if 'A' <= r && r <= 'Z' && !a.game.Guessed().Has(r) {
			tt.Key(0, ui.KeyA+ui.Key(r-'A'))
		}
	}
	if !a.game.IsWon() || cues.wins != 1 || a.session.MatchPoints() == 0 {
		t.Errorf("won %v, cues %+v, points %d", a.game.IsWon(), cues, a.session.MatchPoints())
	}
	if !tt.HasText("The word was " + a.game.Word()) {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestLoadingAWordListToastsWhatItLoaded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "list.txt")
	if err := os.WriteFile(path, []byte("Alpha\nOmega\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Settings{}, Options{PickFile: func(done func(string, error)) { done(ReadFile(path)) }})
	tt := ui.NewTester(a.View, 1000, 800)
	tt.Key(ui.Cmd, ui.KeyO)
	tt.Frame()
	if a.game.Difficulty() != game.NoDifficulty || !tt.HasText("Loaded 2 words. New match!") {
		t.Errorf("texts %q", tt.Texts())
	}
	if !tt.HasText(customListSubtitle) {
		t.Error("the title bar does not say a list of your own is in play")
	}
}

func TestTheSettingsFileFollowsEveryGuess(t *testing.T) {
	a, tt, path := tester(t, settings.Settings{})
	tt.Key(0, wrongKey(a.game))
	saved := settings.Load(path)
	if saved.InFlight == nil || saved.InFlight.Wrong != 1 {
		t.Errorf("saved %+v", saved.InFlight)
	}
	back := New(saved, Options{})
	if back.game.Word() != a.game.Word() || back.game.WrongGuesses() != 1 {
		t.Error("the next launch did not pick the word back up")
	}
}

func wrongKey(g *game.Game) ui.Key {
	return ui.KeyA + ui.Key(wrongLetter(g)-'A')
}
