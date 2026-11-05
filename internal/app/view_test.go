package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/sounds"
	"github.com/zerebos/hangman-mygo/internal/stats"
)

// tester opens the view on a fresh app at the default window size, with the
// settings written to a temporary file.
func tester(t *testing.T, s settings.Settings) (*App, *ui.Tester, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	a := New(s, Options{Save: func(s settings.Settings) { s.Save(path) }})
	return a, ui.NewTester(a.View, 1000, 800), path
}

// recorder is a speaker that keeps what it was asked to play.
type recorder struct{ played [][]float32 }

func (r *recorder) Play(samples []float32, _ time.Duration) { r.played = append(r.played, samples) }

// heard reports whether the recorder played cue.
func (r *recorder) heard(cue []float32) bool {
	for _, s := range r.played {
		if reflect.DeepEqual(s, cue) {
			return true
		}
	}
	return false
}

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
	tt.Key(ui.Cmd, giveUpKey)
	if !a.game.IsWordOver() || a.notice == nil || a.notice.text != gaveUp {
		t.Fatalf("Ctrl+G did not give up: %+v", a.notice)
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
	// The drawer springs open.
	tt.Advance(time.Second)
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
	chooseDifficulty(t, a, tt, "Insane")
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
	chooseDifficulty(t, a, tt, "Insane")
	if a.game.Word() != word || !a.game.Guessed().Has('Q') {
		t.Error("re-clicking the difficulty in play threw the word away")
	}
}

func TestSwitchingMidWordChargesTheWordAndSaysSo(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	word := a.game.Word()
	tt.Key(0, wrongKey(a.game))
	chooseDifficulty(t, a, tt, "Hard")
	if a.session.Stats().WordsLost != 1 || a.session.Stats().ForDifficulty(game.Easy).WordsLost != 1 {
		t.Error("the abandoned word was not charged to Easy")
	}
	if !tt.HasText("Leaving " + word + " counts as a loss in my book.") {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestWinningPlaysTheWinCueAndScores(t *testing.T) {
	cues := &recorder{}
	a := New(settings.Settings{}, Options{Speaker: cues})
	tt := ui.NewTester(a.View, 1000, 800)
	for _, r := range a.game.Word() {
		if 'A' <= r && r <= 'Z' && !a.game.Guessed().Has(r) {
			tt.Key(0, ui.KeyA+ui.Key(r-'A'))
		}
	}
	if !a.game.IsWon() || !cues.heard(sounds.Wood.WordWon()) || a.session.MatchPoints() == 0 {
		t.Errorf("won %v, heard the win %v, points %d", a.game.IsWon(), cues.heard(sounds.Wood.WordWon()), a.session.MatchPoints())
	}
	// The result card rises once the keyboard has folded away.
	tt.Advance(time.Second)
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

// chooseDifficulty opens the difficulty dropdown and chooses label from it.
//
// The dropdown's own button shows the difficulty in play, so its item cannot
// be found by its text. The items are rows of one height in menu order, so
// the target's row is worked out from two items that are not the one in play.
func chooseDifficulty(t *testing.T, a *App, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(subtitle(a.game)); err != nil {
		t.Fatal(err)
	}
	var rows []int
	for i, d := range game.Difficulties {
		if d != a.game.Difficulty() {
			rows = append(rows, i)
		}
	}
	first, ok1 := tt.Find(game.Difficulties[rows[0]].Label())
	second, ok2 := tt.Find(game.Difficulties[rows[1]].Label())
	if !ok1 || !ok2 {
		t.Fatalf("the menu did not open: %q", tt.Texts())
	}
	step := (second.Y - first.Y) / float32(rows[1]-rows[0])
	target := 0
	for i, d := range game.Difficulties {
		if d.Label() == label {
			target = i
		}
	}
	tt.ClickAt(first.X+first.W/2, first.Y+first.H/2+step*float32(target-rows[0]))
	if tt.HasText(game.Difficulties[rows[0]].Label()) && game.Difficulties[rows[0]].Label() != subtitle(a.game) {
		t.Fatalf("the menu is still open after choosing %q", label)
	}
}

// The last word's result shows first; Enter opens the summary, whose numbers
// count up to the match's, and Enter again plays the difficulty again.
func TestAMatchEndsOnItsSummaryAndEnterPlaysItAgain(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	for a.game.WordNumber() < a.game.TotalWords() {
		tt.Key(ui.Cmd, giveUpKey)
		tt.Key(0, ui.KeyEnter)
	}
	tt.Key(ui.Cmd, giveUpKey)
	tt.Advance(time.Second)
	if a.summaryShown || !tt.HasText("See the match") {
		t.Fatalf("the last word skipped its result: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(3 * time.Second)
	if !a.summaryShown || !tt.HasText("You lost the match") || !tt.HasText("10") {
		t.Fatalf("summary %q", tt.Texts())
	}
	if tt.FramePending() {
		t.Error("the summary kept asking for frames once its numbers had counted up")
	}
	tt.Key(0, ui.KeyEnter)
	if a.summaryShown || a.game.IsMatchOver() || a.game.WordNumber() != 1 || a.game.Difficulty() != game.Easy {
		t.Errorf("after Play again: summary %v, word %d", a.summaryShown, a.game.WordNumber())
	}
}

// A won match throws confetti, and the confetti settles too.
func TestAWonMatchThrowsConfettiThatLands(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	g, err := game.FromWordsWithSeed(1, "AB")
	if err != nil {
		t.Fatal(err)
	}
	a.game = g
	tt.Key(0, ui.KeyA)
	tt.Key(0, ui.KeyB)
	tt.Key(0, ui.KeyEnter)
	if !a.summaryShown || len(a.confetti) == 0 {
		t.Fatalf("summary %v, confetti %d", a.summaryShown, len(a.confetti))
	}
	tt.Advance(4 * time.Second)
	if tt.FramePending() {
		t.Error("the confetti never landed")
	}
}

// A list loaded this session can be played again from its summary.
func TestAWordListOfYourOwnCanBePlayedAgain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "list.json")
	if err := os.WriteFile(path, []byte(`{"name":"Pets","words":[{"word":"Cat"},{"word":"Dog"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	a := New(settings.Settings{}, Options{PickFile: func(done func(string, error)) { done(ReadFile(path)) }})
	tt := ui.NewTester(a.View, 1000, 800)
	tt.Key(ui.Cmd, ui.KeyO)
	for !a.game.IsMatchOver() {
		tt.Key(ui.Cmd, giveUpKey)
		// On the last word, Enter opens the summary.
		tt.Key(0, ui.KeyEnter)
	}
	tt.Advance(time.Second)
	if !a.summaryShown || !tt.HasText("Play Pets again") {
		t.Fatalf("summary %v, texts %q", a.summaryShown, tt.Texts())
	}
	tt.Key(0, ui.KeyEnter)
	if a.game.IsMatchOver() || a.game.PackName() != "Pets" || a.game.WordNumber() != 1 || a.session.MatchPoints() != 0 {
		t.Errorf("after Play again: over %v, pack %q, word %d", a.game.IsMatchOver(), a.game.PackName(), a.game.WordNumber())
	}
}

// fakeWindow records what the header's window controls asked of it.
type fakeWindow struct {
	maximized         bool
	minimized, closed int
}

func (w *fakeWindow) Minimize()         { w.minimized++ }
func (w *fakeWindow) Maximize()         { w.maximized = true }
func (w *fakeWindow) Unmaximize()       { w.maximized = false }
func (w *fakeWindow) IsMaximized() bool { return w.maximized }
func (w *fakeWindow) Close()            { w.closed++ }

func TestTheHeadersWindowControlsWorkTheWindow(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	if tt.HasText("Minimize") {
		t.Fatal("controls drawn with no window to control")
	}
	w := &fakeWindow{}
	a.SetWindow(w)
	tt.Frame()
	for _, label := range []string{"Minimize", "Maximize", "Restore", "Close"} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}
	if w.minimized != 1 || w.maximized || w.closed != 1 {
		t.Errorf("window %+v", *w)
	}
}

// Each guess plays its cue: a right letter one step further up the scale
// than the last, a wrong one the knock, and the word's end its phrase. With
// the sound off, nothing plays.
func TestEachGuessPlaysItsCue(t *testing.T) {
	cues := &recorder{}
	a := New(settings.Settings{}, Options{Speaker: cues})
	g, err := game.FromWordsWithSeed(1, "ABC")
	if err != nil {
		t.Fatal(err)
	}
	a.game = g
	tt := ui.NewTester(a.View, 1000, 800)
	tt.Key(0, ui.KeyA)
	tt.Key(0, ui.KeyB)
	tt.Key(0, ui.KeyZ)
	for _, want := range []struct {
		name string
		cue  []float32
	}{
		// ABC is three distinct letters, climbing to the octave on the last.
		{"first right", sounds.Wood.Right(sounds.Top - 2)},
		{"second right", sounds.Wood.Right(sounds.Top - 1)},
		// A list of your own allows six wrong guesses, so the first falls
		// from five semitones up, D3.
		{"first wrong", sounds.Wood.Wrong(sounds.WrongStep(0, a.game.GuessBudget()))},
	} {
		if !cues.heard(want.cue) {
			t.Errorf("did not hear the %s", want.name)
		}
	}
	tt.Key(0, ui.KeyC)
	if !cues.heard(sounds.Wood.Right(sounds.Top)) || !cues.heard(sounds.Wood.WordWon()) {
		t.Error("the last letter did not play its pluck and the win")
	}
	tt.Key(0, ui.KeyEnter)
	if !cues.heard(sounds.Wood.MatchWon()) {
		t.Error("the won match's summary played no phrase")
	}

	before := len(cues.played)
	a.toggleSound()
	fresh, err := game.FromWordsWithSeed(2, "AB")
	if err != nil {
		t.Fatal(err)
	}
	a.game = fresh
	a.freshWord(nil)
	tt.Frame()
	tt.Key(0, ui.KeyA)
	tt.Key(0, ui.KeyZ)
	if len(cues.played) != before {
		t.Errorf("%d cues played with the sound off", len(cues.played)-before)
	}
}

// A hint plays the wrong note it costs and the right note it gives, the
// clue its glint, and a fresh word its own cue.
func TestHintsCluesAndFreshWordsHaveTheirOwnCues(t *testing.T) {
	cues := &recorder{}
	a := New(settings.Settings{}, Options{Speaker: cues})
	g, err := game.FromPackWithSeed(mustPack(t, `{"words": [
		{"word": "Cat", "clue": "It purrs."},
		{"word": "Dog", "clue": "It barks."}
	]}`), 1)
	if err != nil {
		t.Fatal(err)
	}
	a.game = g
	tt := ui.NewTester(a.View, 1000, 800)

	tt.Key(ui.Cmd, ui.KeyH)
	// The first of three letters found, and the first of six misses spent.
	if !cues.heard(sounds.Wood.Hint(sounds.RightStep(0, 3), sounds.WrongStep(0, 6))) {
		t.Error("the hint did not play the cost and the letter")
	}
	tt.Key(ui.Cmd, ui.KeyL)
	if !cues.heard(sounds.Wood.Clue()) {
		t.Error("the clue played no cue")
	}
	if cues.heard(sounds.Wood.NewWord()) {
		t.Fatal("a fresh word's cue played before any word was dealt")
	}
	tt.Key(ui.Cmd, giveUpKey)
	tt.Key(0, ui.KeyEnter)
	if !cues.heard(sounds.Wood.NewWord()) {
		t.Error("the next word played no cue")
	}
}

// Giving up is a loss, and sounds and looks like one.
func TestGivingUpSoundsLikeLosing(t *testing.T) {
	cues := &recorder{}
	a := New(settings.Settings{}, Options{Speaker: cues})
	tt := ui.NewTester(a.View, 1000, 800)
	tt.Key(ui.Cmd, giveUpKey)
	if !cues.heard(sounds.Wood.WordLost()) {
		t.Error("giving up played no loss")
	}
	// And looks like one: the gallows is finished.
	budget := a.game.GuessBudget()
	if _, ok := tt.Find(gallowsLabel(budget, budget)); !ok {
		t.Errorf("the gallows is not finished after giving up: %q", tt.Texts())
	}
}

// A played key is a tile, not a button, so it names itself to assistive
// technology: the letter, and what its guess found.
func TestPlayedKeysSayWhatTheirGuessFound(t *testing.T) {
	a, tt, _ := tester(t, settings.Settings{})
	right, wrong := rune(a.game.Word()[0]), wrongLetter(a.game)
	tt.Key(0, ui.KeyA+ui.Key(right-'A'))
	tt.Key(0, ui.KeyA+ui.Key(wrong-'A'))
	tt.Advance(time.Second)
	for _, label := range []string{playedLabel(right, keyCorrect), playedLabel(wrong, keyWrong)} {
		if _, ok := tt.Find(label); !ok {
			t.Errorf("no %q on screen", label)
		}
	}
	if err := tt.Click("Give up"); err != nil {
		t.Fatal(err)
	}
	if !a.game.IsWordOver() {
		t.Error("Give up did not give up")
	}
}

// A window with no way to ask for a file, as in a browser, offers no Open
// word list anywhere, and one that can offers it on the board and on the
// summary alike.
func TestOpenWordListShowsOnlyWhereFilesCanBeOpened(t *testing.T) {
	pick := func(done func(string, error)) {}
	for _, picker := range []func(func(string, error)){nil, pick} {
		a := New(settings.Settings{}, Options{PickFile: picker})
		tt := ui.NewTester(a.View, 1000, 800)
		offered := picker != nil
		if got := tt.HasText(shortcutOpenWordList.label()); got != offered {
			t.Errorf("picker %v: the legend lists Open a word list: %v", offered, got)
		}
		for a.game.WordNumber() < a.game.TotalWords() {
			tt.Key(ui.Cmd, giveUpKey)
			tt.Key(0, ui.KeyEnter)
		}
		tt.Key(ui.Cmd, giveUpKey)
		tt.Key(0, ui.KeyEnter)
		tt.Advance(3 * time.Second)
		if !a.summaryShown {
			t.Fatalf("picker %v: no summary", offered)
		}
		if got := tt.HasText("Open word list…"); got != offered {
			t.Errorf("picker %v: the summary offers Open word list: %v", offered, got)
		}
	}
}
