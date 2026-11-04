package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
	"github.com/zerebos/hangman-mygo/internal/stats"
	"github.com/zerebos/hangman-mygo/internal/words"
)

func mustWords(t *testing.T, list ...string) *game.Game {
	t.Helper()
	g, err := game.FromWordsWithSeed(1, list...)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// clued is a one-word game whose word has a clue and a category.
func clued(t *testing.T) *game.Game {
	t.Helper()
	g := game.New(game.Easy)
	if err := g.SetPack(mustPack(t, `{"words":[{"word":"Cat","category":"Pets","clue":"It purrs."}]}`)); err != nil {
		t.Fatal(err)
	}
	return g
}

// wrongLetter is a letter not in the word and not guessed yet.
func wrongLetter(g *game.Game) rune {
	for r := 'Z'; r >= 'A'; r-- {
		if !strings.ContainsRune(g.Word(), r) && !g.Guessed().Has(r) {
			return r
		}
	}
	panic("no wrong letter left")
}

func TestPluralSaysSForEverythingButOne(t *testing.T) {
	for n, want := range map[uint64]string{0: "0 words", 1: "1 word", 2: "2 words"} {
		if got := plural(n, "word"); got != want {
			t.Errorf("plural(%d) = %q", n, got)
		}
	}
	if points(1) != "1 point" || points(840) != "840 points" {
		t.Errorf("points: %q, %q", points(1), points(840))
	}
}

func TestTheResetWarningNamesWhatIsLost(t *testing.T) {
	s := stats.Stats{Points: 9210, WordsWon: 31, WordsLost: 9, BestStreak: 11}
	summary := resetStatsSummary(&s)
	// Words played is wins and losses: the whole record goes, not the
	// flattering half of it.
	if !strings.HasPrefix(summary, "9210 points, 40 words and a best streak of 11") || !strings.Contains(summary, "no undo") {
		t.Errorf("summary %q", summary)
	}
}

func TestTheResetWarningDoesNotThreatenAnEmptyTally(t *testing.T) {
	if got := resetStatsSummary(&stats.Stats{}); got != resetNothing {
		t.Errorf("summary %q", got)
	}
	// A streak alone is still something to lose.
	if got := resetStatsSummary(&stats.Stats{MatchesTied: 1}); got == resetNothing {
		t.Errorf("a match count alone was called nothing")
	}
}

func legendOf(g *game.Game, clueShown bool) map[shortcut]bool {
	return legendWith(g, clueShown, false)
}

func legendWith(g *game.Game, clueShown, summaryShown bool) map[shortcut]bool {
	out := map[shortcut]bool{}
	for _, e := range shortcutLegend(g, clueShown, summaryShown, g.Difficulty() != game.NoDifficulty) {
		out[e.shortcut] = e.live
	}
	return out
}

func TestAWordInPlayOffersTheFourStandingShortcuts(t *testing.T) {
	g := clued(t)
	got := legendOf(g, false)
	want := map[shortcut]bool{shortcutHint: true, shortcutClue: true, shortcutChangeWord: true, shortcutOpenWordList: true}
	if len(got) != len(want) {
		t.Fatalf("legend %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%v live %v, want %v", k.label(), got[k], v)
		}
	}
}

// The hint entry greys out on the last guess rather than vanishing.
func TestTheHintEntryGreysOutOnTheLastGuess(t *testing.T) {
	g := mustWords(t, "QUIZ")
	for g.RemainingGuesses() > 1 {
		g.Guess(wrongLetter(g))
	}
	live, listed := legendOf(g, false)[shortcutHint]
	if !listed || live {
		t.Errorf("hint listed %v, live %v", listed, live)
	}
}

func TestAFinishedWordOffersEnterAndDimsWhatItRefuses(t *testing.T) {
	g := mustWords(t, "ALPHA", "OMEGA")
	g.GiveUp()
	got := legendOf(g, false)
	if !got[shortcutNextWord] || got[shortcutHint] || got[shortcutChangeWord] || !got[shortcutOpenWordList] {
		t.Errorf("legend %v", got)
	}
}

func TestAFinishedMatchStopsOfferingEnter(t *testing.T) {
	g := mustWords(t, "ALPHA")
	g.GiveUp()
	if _, listed := legendOf(g, false)[shortcutNextWord]; listed {
		t.Error("Enter offered with no next word to deal")
	}
}

// Enter takes the player from the last word to the summary, and from the
// summary to a new match, and the legend says which.
func TestAFinishedMatchOffersEnterUnderTheNameOfWhatItDoes(t *testing.T) {
	g := game.New(game.Easy)
	for !g.IsMatchOver() {
		g.GiveUp()
		g.NextWord()
	}
	if !legendWith(g, false, false)[shortcutSeeMatch] {
		t.Error("the last word does not offer the summary")
	}
	if got := legendWith(g, false, true); !got[shortcutPlayAgain] || got[shortcutSeeMatch] {
		t.Errorf("the summary offers %v", got)
	}
	// Nothing to replay, nothing offered.
	for _, e := range shortcutLegend(g, false, true, false) {
		if e.shortcut == shortcutPlayAgain {
			t.Error("Play again offered with nothing to replay")
		}
	}
}

func TestAClueIsOfferedOnceAndThenGreyed(t *testing.T) {
	g := clued(t)
	if !canShowClue(g, false) || canShowClue(g, true) {
		t.Error("the clue should be offered until it shows")
	}
	if legendOf(g, true)[shortcutClue] {
		t.Error("the legend still offers a clue that is showing")
	}
}

func TestAWordWithNoClueNeverOffersOne(t *testing.T) {
	g := mustWords(t, "ALPHA")
	if canShowClue(g, false) || clueTooltipFor(g, false) != clueTooltipNone {
		t.Error("a word with no clue offered one")
	}
}

// Once a word is lost the clue is the only thing that explains it.
func TestAClueStaysAvailableAfterTheWordIsOver(t *testing.T) {
	g := clued(t)
	g.GiveUp()
	if !canShowClue(g, false) {
		t.Error("the clue went away with the word")
	}
}

func TestTheClueTooltipSaysWhichStateItIsIn(t *testing.T) {
	g := clued(t)
	if clueTooltipFor(g, false) != clueTooltip || clueTooltipFor(g, true) != clueTooltipShown {
		t.Errorf("tooltips %q, %q", clueTooltipFor(g, false), clueTooltipFor(g, true))
	}
}

func TestTheSubtitleNamesTheDifficultyBeingPlayed(t *testing.T) {
	if got := subtitle(game.New(game.Hard)); got != "Hard" {
		t.Errorf("subtitle %q", got)
	}
}

func TestAPackWithNoNameFallsBackRatherThanShowingABlank(t *testing.T) {
	g := game.New(game.Easy)
	if err := g.SetPack(mustPack(t, `{"name":"   ","words":[{"word":"Cat"}]}`)); err != nil {
		t.Fatal(err)
	}
	if got := subtitle(g); got != customListSubtitle {
		t.Errorf("subtitle %q", got)
	}
	if err := g.SetPack(mustPack(t, `{"name":"Pets","words":[{"word":"Cat"}]}`)); err != nil {
		t.Fatal(err)
	}
	if got := subtitle(g); got != "Pets" {
		t.Errorf("subtitle %q", got)
	}
}

func TestTheLoadedNoticeOwnsUpToPlayingOnlyPartOfAPack(t *testing.T) {
	if got := loadedSummary(200, 10); got != "Loaded 200 words — playing 10 of them." {
		t.Errorf("summary %q", got)
	}
	if got := loadedSummary(1, 1); got != "Loaded 1 word. New match!" {
		t.Errorf("summary %q", got)
	}
}

func TestTheGuessCountMovesOnEveryAcceptedGuessAndHint(t *testing.T) {
	g := mustWords(t, "ALPHABET")
	g.Guess('A')
	g.Guess('A')
	if guessCount(g) != 1 {
		t.Errorf("count %d after a repeat", guessCount(g))
	}
	g.Hint()
	if guessCount(g) != 2 {
		t.Errorf("count %d after a hint", guessCount(g))
	}
}

func TestAGuessedKeySaysWhetherItWasInTheWord(t *testing.T) {
	g := mustWords(t, "ALPHA", "OMEGA")
	g.Guess('A')
	g.Guess('Z')
	if keyStateOf(g, 'A') != keyCorrect || keyStateOf(g, 'Z') != keyWrong || keyStateOf(g, 'B') != keyAvailable {
		t.Error("key states while playing")
	}
	g.GiveUp()
	// The keys never played go out of play; the played ones still show what
	// each guess earned.
	if keyStateOf(g, 'B') != keyOutOfPlay || keyStateOf(g, 'A') != keyCorrect || keyStateOf(g, 'Z') != keyWrong {
		t.Error("key states once the word is over")
	}
}

func TestTheHintTooltipSaysWhyItIsOff(t *testing.T) {
	g := mustWords(t, "QUIZ", "JAZZ")
	if hintTooltipFor(g) != hintTooltip {
		t.Errorf("tooltip %q", hintTooltipFor(g))
	}
	for g.RemainingGuesses() > 1 {
		g.Guess(wrongLetter(g))
	}
	if hintTooltipFor(g) != hintTooltipLastGuess {
		t.Errorf("tooltip %q on the last guess", hintTooltipFor(g))
	}
	g.GiveUp()
	if hintTooltipFor(g) != hintTooltipOver {
		t.Errorf("tooltip %q once over", hintTooltipFor(g))
	}
}

func TestTheMatchSummary(t *testing.T) {
	g := mustWords(t, "ALPHA")
	session := stats.NewSession(stats.Stats{Points: 9999})
	if matchSummary(g, session) != nil {
		t.Error("a summary while the match runs")
	}
	for _, r := range "ALPH" {
		g.Guess(r)
	}
	session.RecordWord(g.Difficulty(), game.Won, g.RemainingGuesses())
	n := matchSummary(g, session)
	// The match's own score, not the lifetime one.
	if n == nil || !n.good || n.text != "Good job, you got 1 out of 1 for 110 points" {
		t.Errorf("summary %+v", n)
	}

	g = mustWords(t, "ALPHA")
	g.GiveUp()
	if n := matchSummary(g, stats.NewSession(stats.Stats{})); n == nil || n.good || !strings.HasPrefix(n.text, "Nice try") {
		t.Errorf("summary %+v", n)
	}
}

func TestAnEvenMatchIsATieAndStillReadsAsGoodNews(t *testing.T) {
	g := mustWords(t, "AB", "AB")
	g.GiveUp()
	g.NextWord()
	g.Guess('A')
	g.Guess('B')
	if n := matchSummary(g, stats.NewSession(stats.Stats{})); n == nil || !n.good || !strings.HasPrefix(n.text, "A tie") {
		t.Errorf("summary %+v", n)
	}
}

func TestARateIsAWholeNumberPercentage(t *testing.T) {
	for rate, want := range map[float32]string{0: "0%", 0.75: "75%", 1: "100%", 2.0 / 3: "67%"} {
		if got := percent(rate); got != want {
			t.Errorf("percent(%v) = %q", rate, got)
		}
	}
}

func TestAnUntouchedWordCostsNothingToWalkAwayFrom(t *testing.T) {
	g := game.New(game.Easy)
	if wordBeingAbandoned(g) != nil {
		t.Error("an untouched word was charged")
	}
	g.Hint()
	if c := wordBeingAbandoned(g); c == nil || c.word != g.Word() || c.difficulty != game.Easy {
		t.Errorf("charge %+v after a hint", c)
	}
}

func TestSwitchingDifficultyChargesTheWordToTheListItCameFrom(t *testing.T) {
	g := game.New(game.Easy)
	word := g.Word()
	g.Guess(wrongLetter(g))
	charge, ok := switchDifficulty(g, game.Insane)
	if !ok || charge == nil || charge.word != word || charge.difficulty != game.Easy {
		t.Errorf("charge %+v, ok %v", charge, ok)
	}
	if g.Difficulty() != game.Insane {
		t.Error("the switch did not happen")
	}
}

func TestReclickingTheDifficultyInPlayCostsTheWordNothing(t *testing.T) {
	g := game.New(game.Easy)
	word := g.Word()
	g.Guess(wrongLetter(g))
	if charge, ok := switchDifficulty(g, game.Easy); ok || charge != nil || g.Word() != word {
		t.Errorf("charge %+v, ok %v", charge, ok)
	}
}

func TestAWordListThatLoadsChargesTheWordItReplaces(t *testing.T) {
	g := game.New(game.Medium)
	g.Guess(wrongLetter(g))
	charge, pack, ok := loadWords(g, "Alpha\nOmega\n", nil)
	if !ok || charge == nil || charge.difficulty != game.Medium || pack == nil || len(pack.List) != 2 {
		t.Errorf("charge %+v, ok %v", charge, ok)
	}
	if g.Difficulty() != game.NoDifficulty || g.PackWords() != 2 {
		t.Error("the list did not load")
	}
}

func TestAWordListThatWillNotLoadChargesNothing(t *testing.T) {
	for name, file := range map[string]struct {
		text string
		err  error
	}{
		"unreadable":    {"", errors.New("gone")},
		"bad JSON":      {`{"words": [`, nil},
		"nothing in it": {"123\n,,,\n", nil},
	} {
		g := game.New(game.Easy)
		word := g.Word()
		g.Guess(wrongLetter(g))
		if charge, pack, ok := loadWords(g, file.text, file.err); ok || charge != nil || pack != nil || g.Word() != word || !g.HasWordToLose() {
			t.Errorf("%s: charge %+v, ok %v", name, charge, ok)
		}
	}
}

func TestASavedMatchIsTheMatchTheWindowOpensOn(t *testing.T) {
	g := game.New(game.Hard)
	g.Guess(wrongLetter(g))
	session := stats.ResumeSession(stats.Stats{}, 260)
	s := settings.Settings{Difficulty: settings.Difficulty(game.Easy), Stats: stats.Stats{Streak: 3}, InFlight: saveMatch(g, session)}

	back, backSession := resumeOrStart(&s)
	if back.Word() != g.Word() || back.WrongGuesses() != 1 || back.Difficulty() != game.Hard {
		t.Errorf("resumed %q, %d wrong, %v", back.Word(), back.WrongGuesses(), back.Difficulty())
	}
	if backSession.MatchPoints() != 260 || backSession.Stats().Streak != 3 {
		t.Errorf("session %d points, streak %d", backSession.MatchPoints(), backSession.Stats().Streak)
	}
}

func TestNoSavedMatchDealsAFreshOneOnTheDifficultyLastPicked(t *testing.T) {
	s := settings.Settings{Difficulty: settings.Difficulty(game.Insane), Stats: stats.Stats{Streak: 2}}
	g, session := resumeOrStart(&s)
	if g.Difficulty() != game.Insane || session.Stats().Streak != 2 || session.MatchPoints() != 0 {
		t.Error("fresh start")
	}
	if g, _ := resumeOrStart(&settings.Settings{}); g.Difficulty() != game.Easy {
		t.Error("a first launch should start on Easy")
	}
}

func TestASavedMatchThatMakesNoSenseCostsOnlyTheMatch(t *testing.T) {
	s := settings.Settings{
		Difficulty: settings.Difficulty(game.Medium),
		Stats:      stats.Stats{Points: 50},
		InFlight:   &settings.SavedMatch{Wrong: 99},
	}
	g, session := resumeOrStart(&s)
	if g.Difficulty() != game.Medium || session.Stats().Points != 50 {
		t.Error("a broken saved match cost more than itself")
	}
}

func TestAMatchThatIsOverIsNotSaved(t *testing.T) {
	g := mustWords(t, "ALPHA")
	g.GiveUp()
	if saveMatch(g, stats.NewSession(stats.Stats{})) != nil {
		t.Error("a finished match was saved")
	}
}

func mustPack(t *testing.T, text string) words.Pack {
	t.Helper()
	p, err := words.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
