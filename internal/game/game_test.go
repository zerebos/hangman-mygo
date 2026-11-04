package game

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zerebos/hangman-mygo/internal/gallows"
	"github.com/zerebos/hangman-mygo/internal/words"
)

// gameWithWord is a game on a single fixed word, so a test never depends on
// which word the seed deals.
func gameWithWord(t *testing.T, word string) *Game {
	t.Helper()
	g, err := FromWordsWithSeed(1, word)
	if err != nil {
		t.Fatalf("FromWordsWithSeed(%q): %v", word, err)
	}
	return g
}

func mustFromWords(t *testing.T, seed uint64, list ...string) *Game {
	t.Helper()
	g, err := FromWordsWithSeed(seed, list...)
	if err != nil {
		t.Fatalf("FromWordsWithSeed(%q): %v", list, err)
	}
	return g
}

// winCurrentGame guesses every letter of the current word.
func winCurrentGame(g *Game) {
	for _, r := range g.Word() {
		if words.IsLetter(r) {
			g.Guess(r)
		}
	}
}

// loseCurrentGame spends the whole budget on letters not in the word.
func loseCurrentGame(g *Game) {
	for _, r := range wrongLetters(g) {
		g.Guess(r)
	}
}

// wrongLetters is as many letters as the budget allows, none of them in the
// current word. The longest bundled word has eleven distinct letters, so
// there are always more than the ten the most generous budget spends.
func wrongLetters(g *Game) []rune {
	var out []rune
	for r := 'A'; r <= 'Z' && len(out) < g.GuessBudget(); r++ {
		if !strings.ContainsRune(g.Word(), r) {
			out = append(out, r)
		}
	}
	return out
}

// packWords is a difficulty's pack as plain uppercase strings.
func packWords(d Difficulty) []string {
	var out []string
	for _, w := range BundledPack(d).Words() {
		out = append(out, w.Word)
	}
	return out
}

// playOut gives up on every word of the match, and returns the words dealt.
func playOut(t *testing.T, g *Game) []string {
	t.Helper()
	seen := []string{g.Word()}
	for !g.IsMatchOver() {
		g.GiveUp()
		if g.NextWord() {
			seen = append(seen, g.Word())
		}
	}
	return seen
}

// finishMatchByGivingUp gives up on every word until the match is over.
func finishMatchByGivingUp(t *testing.T, g *Game) {
	t.Helper()
	for g.GiveUp() == MatchRunning {
		if !g.NextWord() {
			t.Fatal("the match still had words left")
		}
	}
}

func TestEveryBundledPackParses(t *testing.T) {
	for _, d := range Difficulties {
		pack := BundledPack(d)
		if pack.Name != d.Label() {
			t.Errorf("%s pack is named %q", d.Label(), pack.Name)
		}
		if pack.GuessBudget != 0 {
			t.Errorf("%s pack states a budget of %d", d.Label(), pack.GuessBudget)
		}
	}
}

// A bundled pack shorter than a match would quietly end that difficulty's
// matches early.
func TestEveryBundledPackCanFillAWholeMatch(t *testing.T) {
	for _, d := range Difficulties {
		if n := len(packWords(d)); n < MatchWords {
			t.Errorf("%s has only %d words", d.Label(), n)
		}
	}
}

// Two seeds rather than two matches, so the assertion is about the draw and
// not about what the first match happened to remove.
func TestAMatchIsDrawnFromThePackRatherThanBeingAllOfIt(t *testing.T) {
	if len(packWords(Easy)) <= MatchWords {
		t.Fatal("this test only says anything while the pack is bigger than a match")
	}
	drawn := func(seed uint64) []string {
		seen := playOut(t, WithSeed(Easy, seed))
		slices.Sort(seen)
		return seen
	}
	if slices.Equal(drawn(1), drawn(2)) {
		t.Error("two seeds drew the same ten words")
	}
}

// Nothing shown during play may leak the answer. A clue built on the same
// stem as its word is how that actually happened, and six letters is where a
// shared stem stops being a coincidence.
func TestNoBundledClueGivesItsOwnWordAway(t *testing.T) {
	lettersOnly := func(text string) string {
		var b strings.Builder
		for _, r := range text {
			if words.IsLetter(r) {
				b.WriteRune(r)
			}
		}
		return strings.ToLower(b.String())
	}
	for _, d := range Difficulties {
		for _, w := range BundledPack(d).List {
			needle := lettersOnly(w.Word)
			haystack := lettersOnly(w.Category + w.Clue)
			if strings.Contains(haystack, needle) {
				t.Errorf("%s's clue names the word itself", w.Word)
			}
			if len(needle) >= 6 && strings.Contains(haystack, needle[:6]) {
				t.Errorf("%s's clue shares its first six letters", w.Word)
			}
		}
	}
}

// Both are optional in the format, but a bundled word with no clue is a Clue
// button that does nothing for reasons the player cannot see.
func TestEveryBundledWordCarriesACategoryAndAClue(t *testing.T) {
	for _, d := range Difficulties {
		for _, w := range BundledPack(d).Words() {
			if w.Category == "" {
				t.Errorf("%s has no category", w.Word)
			}
			if w.Clue == "" {
				t.Errorf("%s has no clue", w.Word)
			}
		}
	}
}

func TestNoBundledPackRepeatsAWord(t *testing.T) {
	for _, d := range Difficulties {
		list := packWords(d)
		before := len(list)
		slices.Sort(list)
		if len(slices.Compact(list)) != before {
			t.Errorf("%s repeats a word", d.Label())
		}
	}
}

func TestDifficultyWeightsClimbOneStepPerLevel(t *testing.T) {
	var got []int
	for _, d := range Difficulties {
		got = append(got, d.Weight())
	}
	if !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Errorf("weights %v, want 1x to 4x", got)
	}
}

func TestTheEasiestDifficultyIsWorthTheLeast(t *testing.T) {
	if Easy.Weight() != 1 {
		t.Errorf("Easy weight %d", Easy.Weight())
	}
	if Insane.Weight() <= Easy.Weight() {
		t.Error("Insane is not worth more than Easy")
	}
}

// ------------------------------------------------------- the guess budget

func TestEachDifficultyHandsOutItsOwnGuessBudget(t *testing.T) {
	var got []int
	for _, d := range Difficulties {
		got = append(got, d.GuessBudget())
	}
	if !slices.Equal(got, []int{10, 8, 7, 6}) {
		t.Errorf("budgets %v, want easy the most forgiving and insane the original's six", got)
	}
}

func TestTheHardestDifficultyIsTheOriginalGame(t *testing.T) {
	if Insane.GuessBudget() != DefaultGuessBudget {
		t.Errorf("Insane budget %d", Insane.GuessBudget())
	}
	for _, d := range Difficulties {
		if d.GuessBudget() < Insane.GuessBudget() {
			t.Errorf("%s is stingier than Insane", d.Label())
		}
	}
}

// Only a budget the gallows can draw gets one new part per wrong guess.
// Outside it a stage would draw two parts or repeat one.
func TestEveryBudgetBuysExactlyOneBodyPartPerWrongGuess(t *testing.T) {
	for _, d := range Difficulties {
		budget := d.GuessBudget()
		if budget < gallows.CoreParts || budget > len(gallows.Parts) {
			t.Errorf("%s's budget of %d is outside what the gallows draws", d.Label(), budget)
		}
		for wrong := 0; wrong <= budget; wrong++ {
			if got := gallows.PartsDrawn(budget, wrong); got != wrong {
				t.Errorf("%s at %d draws %d parts", d.Label(), wrong, got)
			}
		}
	}
}

func TestAGameLastsExactlyItsDifficultysBudgetAndNotAGuessLess(t *testing.T) {
	for _, d := range Difficulties {
		label := d.Label()
		budget := d.GuessBudget()
		g := WithSeed(d, 7)
		if g.GuessBudget() != budget || g.RemainingGuesses() != budget {
			t.Fatalf("%s starts with budget %d and %d left", label, g.GuessBudget(), g.RemainingGuesses())
		}
		for spent, letter := range wrongLetters(g) {
			if g.IsWordOver() {
				t.Fatalf("%s was over after %d", label, spent)
			}
			if out := g.Guess(letter); out.Result != Wrong {
				t.Fatalf("%s: %c was %v", label, letter, out.Result)
			}
			if g.WrongGuesses() != spent+1 || g.RemainingGuesses() != budget-spent-1 {
				t.Fatalf("%s after %d: wrong %d, left %d", label, spent+1, g.WrongGuesses(), g.RemainingGuesses())
			}
		}
		if !g.IsWordOver() {
			t.Errorf("%s survived its whole budget", label)
		}
		if g.Result() != Lost || g.RemainingGuesses() != 0 {
			t.Errorf("%s: result %v with %d left", label, g.Result(), g.RemainingGuesses())
		}
	}
}

func TestAWordListOfYourOwnIsPlayedByTheOriginalsRules(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	if g.Difficulty() != NoDifficulty {
		t.Errorf("difficulty %v", g.Difficulty())
	}
	if g.GuessBudget() != DefaultGuessBudget || g.RemainingGuesses() != DefaultGuessBudget {
		t.Errorf("budget %d, left %d", g.GuessBudget(), g.RemainingGuesses())
	}
}

func TestChangingDifficultyMidMatchChangesTheBudgetWithIt(t *testing.T) {
	g := WithSeed(Insane, 5)
	if g.GuessBudget() != 6 {
		t.Fatalf("budget %d", g.GuessBudget())
	}
	g.Guess(wrongLetters(g)[0])
	if g.RemainingGuesses() != 5 {
		t.Fatalf("left %d", g.RemainingGuesses())
	}

	g.SetDifficulty(Easy)
	if g.GuessBudget() != 10 {
		t.Errorf("Easy budget %d", g.GuessBudget())
	}
	if g.RemainingGuesses() != 10 {
		t.Errorf("the new word should start fresh, %d left", g.RemainingGuesses())
	}

	g.SetDifficulty(Hard)
	if g.GuessBudget() != 7 {
		t.Errorf("Hard budget %d", g.GuessBudget())
	}

	if err := g.SetWordList("BANANA"); err != nil {
		t.Fatal(err)
	}
	if g.GuessBudget() != DefaultGuessBudget {
		t.Errorf("word list budget %d", g.GuessBudget())
	}
}

func TestARejectedWordListLeavesTheBudgetAlone(t *testing.T) {
	g := WithSeed(Easy, 2)
	if err := g.SetWordList("   "); err == nil {
		t.Fatal("a blank list was accepted")
	}
	if g.GuessBudget() != Easy.GuessBudget() {
		t.Errorf("budget %d", g.GuessBudget())
	}
}

func TestCorrectGuessRevealsEveryOccurrence(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	if g.Display() != "______" {
		t.Fatalf("display %q", g.Display())
	}
	if out := g.Guess('A'); out.Result != Correct {
		t.Errorf("result %v", out.Result)
	}
	if g.Display() != "_A_A_A" || g.WrongGuesses() != 0 {
		t.Errorf("display %q, wrong %d", g.Display(), g.WrongGuesses())
	}
}

func TestWrongGuessCostsOneGuess(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	if out := g.Guess('Z'); out.Result != Wrong {
		t.Errorf("result %v", out.Result)
	}
	if g.WrongGuesses() != 1 || g.RemainingGuesses() != DefaultGuessBudget-1 {
		t.Errorf("wrong %d, left %d", g.WrongGuesses(), g.RemainingGuesses())
	}
	if g.Display() != "______" {
		t.Errorf("display %q", g.Display())
	}
}

func TestDuplicateGuessIsFree(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	g.Guess('Z')
	if out := g.Guess('Z'); out.Result != Duplicate {
		t.Errorf("second Z was %v", out.Result)
	}
	if g.WrongGuesses() != 1 {
		t.Errorf("wrong %d", g.WrongGuesses())
	}
	g.Guess('A')
	if out := g.Guess('A'); out.Result != Duplicate {
		t.Errorf("second A was %v", out.Result)
	}
	if g.WrongGuesses() != 1 {
		t.Errorf("wrong %d", g.WrongGuesses())
	}
}

func TestInvalidGuessIsFree(t *testing.T) {
	g := gameWithWord(t, "ADD/DROP FORM")
	for _, bad := range []rune{'1', '/', ' ', '!', rune(0xE9)} {
		if out := g.Guess(bad); out.Result != Invalid {
			t.Errorf("%q was %v", bad, out.Result)
		}
	}
	if g.WrongGuesses() != 0 || g.Guessed() != 0 {
		t.Errorf("wrong %d, guessed %q", g.WrongGuesses(), g.Guessed())
	}
}

func TestGuessesAreCaseInsensitive(t *testing.T) {
	g := gameWithWord(t, "Banana")
	if g.Word() != "BANANA" {
		t.Errorf("word %q", g.Word())
	}
	if out := g.Guess('a'); out.Result != Correct {
		t.Errorf("a was %v", out.Result)
	}
	if out := g.Guess('A'); out.Result != Duplicate {
		t.Errorf("A was %v", out.Result)
	}
	if g.Display() != "_A_A_A" {
		t.Errorf("display %q", g.Display())
	}
}

func TestSpacesAndSlashesAreFree(t *testing.T) {
	g := gameWithWord(t, "Add/Drop Form")
	if g.Display() != "___/____ ____" {
		t.Errorf("display %q", g.Display())
	}
	for _, r := range "ADROPFM" {
		g.Guess(r)
	}
	if !g.IsWon() {
		t.Errorf("not won, display %q", g.Display())
	}
	if g.WrongGuesses() != 0 {
		t.Errorf("wrong %d", g.WrongGuesses())
	}
}

func TestSixWrongGuessesLoseTheGame(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	for _, r := range "CDEFG" {
		out := g.Guess(r)
		if out.Result != Wrong || out.Word != InPlay || g.IsWordOver() {
			t.Fatalf("%c: %+v", r, out)
		}
	}
	out := g.Guess('H')
	if g.WrongGuesses() != DefaultGuessBudget || out.Word != Lost {
		t.Errorf("wrong %d, outcome %+v", g.WrongGuesses(), out)
	}
	if !g.IsWordOver() || g.IsWon() {
		t.Errorf("over %v, won %v", g.IsWordOver(), g.IsWon())
	}
	if g.WordsLost() != 1 || g.WordsWon() != 0 {
		t.Errorf("won %d, lost %d", g.WordsWon(), g.WordsLost())
	}
	if g.Display() != "BANANA" {
		t.Errorf("a lost word should be revealed, display %q", g.Display())
	}
}

func TestGuessingAllLettersWinsTheGame(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	if g.Guess('B').Word != InPlay || g.Guess('A').Word != InPlay {
		t.Fatal("the word ended early")
	}
	out := g.Guess('N')
	if out.Result != Correct || out.Word != Won {
		t.Errorf("outcome %+v", out)
	}
	if !g.IsWon() || g.WordsWon() != 1 || g.WordsLost() != 0 {
		t.Errorf("won %v, %d won, %d lost", g.IsWon(), g.WordsWon(), g.WordsLost())
	}
	if g.Display() != "BANANA" {
		t.Errorf("display %q", g.Display())
	}
}

func TestGuessingAfterGameOverIsANoOp(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	winCurrentGame(g)
	if !g.IsWordOver() {
		t.Fatal("the word is not over")
	}
	if out := g.Guess('Z'); out.Result != Ignored {
		t.Errorf("result %v", out.Result)
	}
	if g.WrongGuesses() != 0 || g.WordsWon() != 1 {
		t.Errorf("wrong %d, won %d", g.WrongGuesses(), g.WordsWon())
	}
}

func TestGiveUpCountsAsALoss(t *testing.T) {
	g := mustFromWords(t, 7, "BANANA", "APPLE")
	if out := g.GiveUp(); out != MatchRunning {
		t.Errorf("giving up on the first of two words ended the match as %v", out)
	}
	if !g.IsWordOver() || g.IsWon() || g.WordsLost() != 1 || g.Result() != Lost {
		t.Errorf("over %v, won %v, lost %d, result %v", g.IsWordOver(), g.IsWon(), g.WordsLost(), g.Result())
	}
	// It spends the rest of the budget, so it is lost as any word is.
	if g.WrongGuesses() != g.GuessBudget() || g.RemainingGuesses() != 0 {
		t.Errorf("gave up with %d of %d wrong", g.WrongGuesses(), g.GuessBudget())
	}
	// Giving up twice must not double-count.
	if out := g.GiveUp(); out != MatchRunning {
		t.Errorf("second give up returned %v", out)
	}
	if g.WordsLost() != 1 {
		t.Errorf("lost %d", g.WordsLost())
	}
}

func TestAvailableLettersShrinkAsLettersAreGuessed(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	if n := len(g.AvailableLetters()); n != 26 {
		t.Errorf("%d available", n)
	}
	g.Guess('A')
	g.Guess('Z')
	if n := len(g.AvailableLetters()); n != 24 {
		t.Errorf("%d available", n)
	}
	if slices.Contains(g.AvailableLetters(), 'A') {
		t.Error("A is still available")
	}
	if got := g.Guessed().String(); got != "AZ" {
		t.Errorf("guessed %q", got)
	}
}

func TestNewGameDealsAFreshWordAndClearsState(t *testing.T) {
	g := mustFromWords(t, 3, "BANANA", "APPLE")
	g.Guess('Z')
	if g.NextWord() {
		t.Fatal("cannot skip a word that is still in play")
	}
	g.GiveUp()
	if !g.NextWord() {
		t.Fatal("no next word")
	}
	if g.WrongGuesses() != 0 || g.Guessed() != 0 || g.IsWordOver() {
		t.Errorf("wrong %d, guessed %q, over %v", g.WrongGuesses(), g.Guessed(), g.IsWordOver())
	}
	if g.WordNumber() != 2 {
		t.Errorf("word number %d", g.WordNumber())
	}
}

// Blank and absent have to be the same state by the time anything reads it,
// or the view's fallback wording never gets its turn and the title bar shows
// three spaces.
func TestAPackThatNamesItselfNothingIsAPackWithNoName(t *testing.T) {
	for _, name := range []string{"", "   ", "\t\n"} {
		pack, err := words.Parse(fmt.Sprintf(`{ "name": %q, "words": [{ "word": "Alpha" }] }`, name))
		if err != nil {
			t.Fatal(err)
		}
		g, err := FromPack(pack)
		if err != nil {
			t.Fatal(err)
		}
		if g.PackName() != "" {
			t.Errorf("%q was taken as the name %q", name, g.PackName())
		}
	}
	pack, err := words.Parse(`{ "name": "  Pets  ", "words": [{ "word": "Alpha" }] }`)
	if err != nil {
		t.Fatal(err)
	}
	g, err := FromPack(pack)
	if err != nil {
		t.Fatal(err)
	}
	if g.PackName() != "Pets" {
		t.Errorf("name %q", g.PackName())
	}
}

func TestAMatchNeverRepeatsAWord(t *testing.T) {
	pack := packWords(Easy)
	seen := playOut(t, WithSeed(Easy, 42))
	if len(seen) != MatchWords {
		t.Fatalf("%d words dealt", len(seen))
	}
	sorted := slices.Clone(seen)
	slices.Sort(sorted)
	if len(slices.Compact(sorted)) != len(seen) {
		t.Errorf("a word was dealt twice: %v", seen)
	}
	for _, w := range seen {
		if !slices.Contains(pack, w) {
			t.Errorf("%s is not in the pack", w)
		}
	}
}

// All ten words rather than the first, because the first is dealt by the
// draw and the other nine by the deals after it. A second seed rules out a
// match that every seed deals the same.
func TestTheSameSeedDrawsTheSameMatch(t *testing.T) {
	first := playOut(t, WithSeed(Easy, 7))
	if len(first) != MatchWords {
		t.Fatalf("%d words dealt", len(first))
	}
	if again := playOut(t, WithSeed(Easy, 7)); !slices.Equal(first, again) {
		t.Errorf("seed 7 dealt %v then %v", first, again)
	}
	if other := playOut(t, WithSeed(Easy, 8)); slices.Equal(first, other) {
		t.Errorf("seeds 7 and 8 both dealt %v", first)
	}
}

func TestAPackSmallerThanAMatchIsPlayedInFull(t *testing.T) {
	g, err := FromWords("ALPHA", "OMEGA")
	if err != nil {
		t.Fatal(err)
	}
	if g.TotalWords() != 2 {
		t.Errorf("total %d", g.TotalWords())
	}
}

func TestALoadedPackMayAskForItsOwnGuessBudget(t *testing.T) {
	pack, err := words.Parse(`{ "guess_budget": 9, "words": [{ "word": "Alpha" }] }`)
	if err != nil {
		t.Fatal(err)
	}
	g, err := FromPack(pack)
	if err != nil {
		t.Fatal(err)
	}
	if g.GuessBudget() != 9 {
		t.Errorf("budget %d", g.GuessBudget())
	}
}

// Pulled rather than refused: the nearest playable budget is better than an
// error the player cannot act on.
func TestABudgetTheGallowsCannotDrawIsPulledToOneItCan(t *testing.T) {
	for _, c := range []struct{ asked, want int }{
		{0, gallows.CoreParts},
		{2, gallows.CoreParts},
		{40, len(gallows.Parts)},
	} {
		pack := words.FromWords("ALPHA")
		pack.GuessBudget = c.asked
		g, err := FromPack(pack)
		if err != nil {
			t.Fatal(err)
		}
		if g.GuessBudget() != c.want {
			t.Errorf("asked for %d, got %d", c.asked, g.GuessBudget())
		}
	}
}

// The bundled packs state no budget, so this guards the rule rather than any
// file we ship.
func TestABundledDifficultyDoesNotLetAPackArgueAboutTheBudget(t *testing.T) {
	if got := budgetFor(Insane, 10); got != Insane.GuessBudget() {
		t.Errorf("Insane with a pack asking for 10 got %d", got)
	}
	if got := budgetFor(NoDifficulty, 10); got != 10 {
		t.Errorf("a loaded pack asking for 10 got %d", got)
	}
}

func TestMatchEndsAfterTheLastWord(t *testing.T) {
	g := WithSeed(Hard, 99)
	if g.TotalWords() != 10 {
		t.Fatalf("total %d", g.TotalWords())
	}
	for word := 1; word <= 10; word++ {
		if g.WordNumber() != word {
			t.Fatalf("word number %d, want %d", g.WordNumber(), word)
		}
		if g.IsMatchOver() {
			t.Fatalf("match ended early at word %d", word)
		}
		winCurrentGame(g)
		g.NextWord()
	}
	if !g.IsMatchOver() || g.WordsWon() != 10 {
		t.Errorf("over %v, won %d", g.IsMatchOver(), g.WordsWon())
	}
	if g.NextWord() {
		t.Error("a word was dealt after the match ended")
	}
}

// The Java original skipped the per-word event on the last word. The port
// reports both.
func TestFinalGameReportsBothTheGameAndTheMatchResult(t *testing.T) {
	g := mustFromWords(t, 5, "CAT")
	g.Guess('C')
	g.Guess('A')
	out := g.Guess('T')
	if out != (GuessOutcome{Result: Correct, Word: Won, Match: MatchWon}) {
		t.Errorf("outcome %+v", out)
	}
	if !g.IsMatchOver() {
		t.Error("the match is not over")
	}
}

func TestMatchOutcomeIsWinWhenWinsExceedLosses(t *testing.T) {
	g := mustFromWords(t, 11, "ONE", "TWO", "SIX")
	winCurrentGame(g)
	g.NextWord()
	winCurrentGame(g)
	g.NextWord()
	loseCurrentGame(g)
	if g.MatchOutcome() != MatchWon || g.WordsWon() != 2 || g.WordsLost() != 1 {
		t.Errorf("outcome %v, %d-%d", g.MatchOutcome(), g.WordsWon(), g.WordsLost())
	}
}

func TestMatchOutcomeIsLossWhenLossesExceedWins(t *testing.T) {
	g := mustFromWords(t, 11, "ONE", "TWO", "SIX")
	loseCurrentGame(g)
	g.NextWord()
	loseCurrentGame(g)
	g.NextWord()
	winCurrentGame(g)
	if g.MatchOutcome() != MatchLost || g.WordsWon() != 1 || g.WordsLost() != 2 {
		t.Errorf("outcome %v, %d-%d", g.MatchOutcome(), g.WordsWon(), g.WordsLost())
	}
}

func TestMatchOutcomeIsTieWhenWinsEqualLosses(t *testing.T) {
	g := mustFromWords(t, 11, "ONE", "TWO")
	winCurrentGame(g)
	g.NextWord()
	if out := g.GiveUp(); out != MatchTied {
		t.Errorf("give up returned %v", out)
	}
	if g.MatchOutcome() != MatchTied {
		t.Errorf("outcome %v", g.MatchOutcome())
	}
}

func TestGivingUpOnTheLastWordEndsTheMatch(t *testing.T) {
	g := mustFromWords(t, 5, "CAT")
	if out := g.GiveUp(); out != MatchLost {
		t.Errorf("give up returned %v", out)
	}
	if !g.IsMatchOver() {
		t.Error("the match is not over")
	}
}

// Picking a difficulty before playing a letter is choosing where to start,
// not abandoning anything.
func TestAnUntouchedWordIsFreeToWalkAwayFrom(t *testing.T) {
	if WithSeed(Insane, 3).HasWordToLose() {
		t.Error("an untouched word has something to lose")
	}
}

func TestOneGuessIsEnoughToMakeAWordWorthLosing(t *testing.T) {
	g := WithSeed(Insane, 3)
	g.Guess(wrongLetters(g)[0])
	if !g.HasWordToLose() {
		t.Error("a wrong guess left nothing to lose")
	}

	g = WithSeed(Insane, 3)
	hit := []rune(g.Word())[0]
	if out := g.Guess(hit); out.Result != Correct {
		t.Fatalf("%c was %v", hit, out.Result)
	}
	if !g.HasWordToLose() {
		t.Error("a right guess left nothing to lose")
	}
}

// A hint records its letter in the same set a guess does, so a word only
// ever hinted at is still a word in progress.
func TestAHintMakesAWordWorthLosingToo(t *testing.T) {
	g := WithSeed(Easy, 4)
	if g.HasWordToLose() {
		t.Fatal("a fresh word has something to lose")
	}
	if out := g.Hint(); out.Result != Revealed {
		t.Fatalf("hint %+v", out)
	}
	if !g.HasWordToLose() {
		t.Error("a hint left nothing to lose")
	}
}

// Won and lost words are already scored, so walking away from one must not
// charge for it twice.
func TestAFinishedWordIsNotAWordToLose(t *testing.T) {
	g := gameWithWord(t, "BANANA")
	winCurrentGame(g)
	if g.Result() != Won || g.HasWordToLose() {
		t.Errorf("won word: result %v, to lose %v", g.Result(), g.HasWordToLose())
	}

	g = WithSeed(Insane, 3)
	loseCurrentGame(g)
	if g.Result() != Lost || g.HasWordToLose() {
		t.Errorf("lost word: result %v, to lose %v", g.Result(), g.HasWordToLose())
	}
}

// The two have to agree, because the UI asks the predicate and then acts on
// the method.
func TestWouldSwitchToAnswersForSetDifficulty(t *testing.T) {
	g := WithSeed(Hard, 6)
	if g.WouldSwitchTo(Hard) {
		t.Error("the difficulty in play would switch")
	}
	for _, d := range Difficulties {
		c := WithSeed(Hard, 6)
		if would, did := c.WouldSwitchTo(d), c.SetDifficulty(d); would != did {
			t.Errorf("%s disagreed: would %v, did %v", d.Label(), would, did)
		}
	}

	// Over: every pill switches, the one just played included.
	finishMatchByGivingUp(t, g)
	for _, d := range Difficulties {
		if !g.WouldSwitchTo(d) {
			t.Errorf("%s should restart a finished match", d.Label())
		}
	}

	// A custom list belongs to no difficulty, so all four switch.
	g = gameWithWord(t, "BANANA")
	for _, d := range Difficulties {
		if !g.WouldSwitchTo(d) {
			t.Errorf("%s should switch out of a custom list", d.Label())
		}
	}
}

// The pills fire even when already selected, so a stray click on the
// difficulty in play must not cost the word in hand.
func TestReselectingTheCurrentDifficultyLeavesTheGameAlone(t *testing.T) {
	g := WithSeed(Insane, 3)
	word := g.Word()
	spend := wrongLetters(g)[0]
	g.Guess(spend)

	if g.SetDifficulty(Insane) {
		t.Error("reselecting the difficulty in play should refuse")
	}
	if g.Word() != word {
		t.Errorf("word %q, want %q", g.Word(), word)
	}
	if g.WrongGuesses() != 1 || !g.Guessed().Has(spend) {
		t.Errorf("wrong %d, guessed %q", g.WrongGuesses(), g.Guessed())
	}
	if g.WordNumber() != 1 {
		t.Errorf("word number %d", g.WordNumber())
	}
}

func TestPickingADifferentDifficultyStillStartsANewGame(t *testing.T) {
	g := WithSeed(Insane, 3)
	g.Guess(wrongLetters(g)[0])

	if !g.SetDifficulty(Easy) {
		t.Fatal("a different difficulty should restart")
	}
	if g.Difficulty() != Easy || g.WrongGuesses() != 0 || g.Guessed() != 0 {
		t.Errorf("difficulty %v, wrong %d, guessed %q", g.Difficulty(), g.WrongGuesses(), g.Guessed())
	}
	if g.GuessBudget() != Easy.GuessBudget() {
		t.Errorf("budget %d", g.GuessBudget())
	}
	if !slices.Contains(packWords(Easy), g.Word()) {
		t.Errorf("%s is not an Easy word", g.Word())
	}
}

// At the end of a match the footer says to pick a difficulty, and the one
// just played is one of the four on offer, so the no-op must not eat that
// click.
func TestReselectingTheCurrentDifficultyRestartsOnceTheMatchIsOver(t *testing.T) {
	g := WithSeed(Insane, 3)
	finishMatchByGivingUp(t, g)
	if !g.IsMatchOver() {
		t.Fatal("the match is not over")
	}
	if !g.SetDifficulty(Insane) {
		t.Fatal("the same difficulty should replay a finished match")
	}
	if g.IsMatchOver() || g.WordsWon() != 0 || g.WordsLost() != 0 || g.WordNumber() != 1 {
		t.Errorf("over %v, %d-%d, word %d", g.IsMatchOver(), g.WordsWon(), g.WordsLost(), g.WordNumber())
	}
}

func TestEveryPillRestartsOutOfACustomWordList(t *testing.T) {
	for _, d := range Difficulties {
		g := gameWithWord(t, "BANANA")
		if g.Difficulty() != NoDifficulty {
			t.Fatalf("difficulty %v", g.Difficulty())
		}
		if !g.SetDifficulty(d) {
			t.Errorf("%s should restart out of a custom list", d.Label())
		}
		if g.Difficulty() != d {
			t.Errorf("difficulty %v, want %v", g.Difficulty(), d)
		}
	}
}

func TestSwitchingDifficultyStartsAFreshMatch(t *testing.T) {
	g := WithSeed(Easy, 1)
	loseCurrentGame(g)
	if g.WordsLost() != 1 {
		t.Fatalf("lost %d", g.WordsLost())
	}
	g.SetDifficulty(Insane)
	if g.Difficulty() != Insane || g.WordsWon() != 0 || g.WordsLost() != 0 {
		t.Errorf("difficulty %v, %d-%d", g.Difficulty(), g.WordsWon(), g.WordsLost())
	}
	if g.IsWordOver() || g.IsMatchOver() || g.WordNumber() != 1 {
		t.Errorf("word over %v, match over %v, word %d", g.IsWordOver(), g.IsMatchOver(), g.WordNumber())
	}
	if !slices.Contains(packWords(Insane), g.Word()) {
		t.Errorf("%s is not an Insane word", g.Word())
	}
}

func TestACustomWordListHasNoDifficulty(t *testing.T) {
	g, err := FromWords("Hello World")
	if err != nil {
		t.Fatal(err)
	}
	if g.Difficulty() != NoDifficulty || g.Word() != "HELLO WORLD" || g.Display() != "_____ _____" {
		t.Errorf("difficulty %v, word %q, display %q", g.Difficulty(), g.Word(), g.Display())
	}
}

func TestUnplayableWordListsAreRejected(t *testing.T) {
	if _, err := FromWords(); err == nil {
		t.Error("an empty list was accepted")
	}
	if _, err := FromWords("", "   ", "42 / 7"); err == nil {
		t.Error("a list of junk was accepted")
	}
}

func TestSetWordListRejectsEmptyListsWithoutDisturbingTheMatch(t *testing.T) {
	g := WithSeed(Easy, 1)
	word := g.Word()
	if err := g.SetWordList("  ", "123"); !errors.Is(err, ErrEmptyWordList) {
		t.Errorf("error %v", err)
	}
	if g.Word() != word || g.Difficulty() != Easy {
		t.Errorf("word %q, difficulty %v", g.Word(), g.Difficulty())
	}

	if err := g.SetWordList("Rustacean"); err != nil {
		t.Fatal(err)
	}
	if g.Word() != "RUSTACEAN" || g.Difficulty() != NoDifficulty {
		t.Errorf("word %q, difficulty %v", g.Word(), g.Difficulty())
	}
}

func TestSeededGamesAreReproducible(t *testing.T) {
	if a, b := WithSeed(Medium, 2024), WithSeed(Medium, 2024); a.Word() != b.Word() {
		t.Errorf("%q and %q", a.Word(), b.Word())
	}
}

// ------------------------------------------------------------------ hints

func TestAHintRevealsExactlyOneNewLetter(t *testing.T) {
	g := gameWithWord(t, "ALPHABET")
	g.Guess('A')
	if g.Display() != "A___A___" {
		t.Fatalf("display %q", g.Display())
	}
	out := g.Hint()
	if out.Result != Revealed {
		t.Fatalf("a fresh game should have a hint to give, got %+v", out)
	}
	if !strings.ContainsRune(g.Word(), out.Letter) {
		t.Errorf("%c is not in the word", out.Letter)
	}
	if out.Letter == 'A' {
		t.Error("a hint may not re-reveal a guessed letter")
	}
	if g.Guessed().Len() != 2 || !g.Guessed().Has(out.Letter) {
		t.Errorf("guessed %q, a hint adds one letter and no more", g.Guessed())
	}
	if !strings.ContainsRune(g.Display(), out.Letter) {
		t.Errorf("display %q", g.Display())
	}
}

func TestAHintCostsExactlyOneWrongGuess(t *testing.T) {
	g := gameWithWord(t, "ALPHABET")
	budget := g.GuessBudget()
	if out := g.Hint(); out.Result != Revealed {
		t.Fatalf("hint %+v", out)
	}
	if g.WrongGuesses() != 1 || g.RemainingGuesses() != budget-1 {
		t.Errorf("wrong %d, left %d", g.WrongGuesses(), g.RemainingGuesses())
	}
	if g.IsWordOver() {
		t.Error("one hint ended a fresh word")
	}
}

func TestAHintEndsNeitherTheGameNorTheMatch(t *testing.T) {
	g := gameWithWord(t, "ALPHABET")
	if out := g.Hint(); out.Result != Revealed {
		t.Fatalf("hint %+v", out)
	}
	if g.WordsWon() != 0 || g.WordsLost() != 0 || g.MatchOutcome() != MatchRunning || g.Result() != InPlay {
		t.Errorf("%d-%d, match %v, word %v", g.WordsWon(), g.WordsLost(), g.MatchOutcome(), g.Result())
	}
}

func TestASeededGameHintsTheSameLetterTwice(t *testing.T) {
	hintOf := func(seed uint64) HintOutcome {
		return mustFromWords(t, seed, "ALPHABET").Hint()
	}
	if a, b := hintOf(7), hintOf(7); a != b {
		t.Errorf("%+v then %+v", a, b)
	}
}

func TestAHintIsRefusedWithOneGuessLeft(t *testing.T) {
	g := gameWithWord(t, "ALPHABET")
	for _, r := range wrongLetters(g)[:g.GuessBudget()-1] {
		g.Guess(r)
	}
	if g.RemainingGuesses() != 1 || g.IsWordOver() {
		t.Fatalf("left %d, over %v", g.RemainingGuesses(), g.IsWordOver())
	}

	guessed, wrong := g.Guessed(), g.WrongGuesses()
	if g.CanHint() {
		t.Error("the last guess is not a hint to spend")
	}
	if out := g.Hint(); out.Result != NoGuessToSpare {
		t.Errorf("hint %+v", out)
	}
	if g.Guessed() != guessed || g.WrongGuesses() != wrong {
		t.Errorf("a refused hint revealed or charged something: guessed %q, wrong %d", g.Guessed(), g.WrongGuesses())
	}
	if g.IsWordOver() {
		t.Error("a refused hint lost the word")
	}
}

func TestAHintIsAvailableRightUpToThatPoint(t *testing.T) {
	g := gameWithWord(t, "ALPHABET")
	for _, r := range wrongLetters(g)[:g.GuessBudget()-2] {
		g.Guess(r)
	}
	if g.RemainingGuesses() != 2 {
		t.Fatalf("left %d", g.RemainingGuesses())
	}
	if !g.CanHint() {
		t.Error("no hint with two guesses left")
	}
	if out := g.Hint(); out.Result != Revealed {
		t.Errorf("hint %+v", out)
	}
	if g.RemainingGuesses() != 1 {
		t.Errorf("left %d", g.RemainingGuesses())
	}
}

func TestAHintIsRefusedOnceTheGameIsOver(t *testing.T) {
	lost := gameWithWord(t, "ALPHABET")
	loseCurrentGame(lost)
	if lost.CanHint() || lost.Hint().Result != HintIgnored {
		t.Error("a lost word gave a hint")
	}

	won := gameWithWord(t, "ALPHABET")
	winCurrentGame(won)
	if won.CanHint() || won.Hint().Result != HintIgnored {
		t.Error("a won word gave a hint")
	}
}

func TestAHintIsRefusedWhenTheWordIsFullyRevealed(t *testing.T) {
	g := gameWithWord(t, "ALPHABET")
	winCurrentGame(g)
	for _, c := range g.Cells() {
		if !c.Revealed {
			t.Fatalf("cell %q is hidden", c.Value)
		}
	}
	if g.CanHint() {
		t.Error("CanHint on a revealed word")
	}
	if out := g.Hint(); out.Result != HintIgnored {
		t.Errorf("hint %+v", out)
	}
	if g.WrongGuesses() != 0 {
		t.Errorf("a refused hint charged %d", g.WrongGuesses())
	}
}

// The letter that finishes a word is the player's to find: with one letter
// left, a hint is refused and changes nothing.
func TestAHintNeverRevealsTheLastLetter(t *testing.T) {
	g := gameWithWord(t, "CAT")
	g.Guess('C')
	g.Guess('A')
	if g.CanHint() {
		t.Error("a hint is offered for the last letter")
	}
	if out := g.Hint(); out != (HintOutcome{Result: LastLetter}) {
		t.Errorf("hint %+v", out)
	}
	if g.Display() != "CA_" || g.WrongGuesses() != 0 || g.IsWordOver() {
		t.Errorf("display %q, wrong %d, over %v", g.Display(), g.WrongGuesses(), g.IsWordOver())
	}
}

// With two letters left a hint reveals one of them, and then refuses the
// other: no number of hints in a row can finish a word.
func TestHintsInARowStopShortOfTheWord(t *testing.T) {
	g := gameWithWord(t, "CAT")
	g.Guess('C')
	if out := g.Hint(); out.Result != Revealed || g.IsWordOver() {
		t.Fatalf("hint %+v", out)
	}
	if out := g.Hint(); out.Result != LastLetter {
		t.Errorf("second hint %+v", out)
	}
	if g.IsWordOver() {
		t.Error("hints finished the word")
	}
}

// ----------------------------------------------------- the match in flight

// matchInFlight is a three-word match with a guess and a hint spent on the
// word in hand. All three words carry the same category and clue, so a test
// about what survives a round trip does not depend on which is dealt first.
func matchInFlight(t *testing.T) *Game {
	t.Helper()
	var list []words.Word
	for _, w := range []string{"Laptop", "Bagel", "Kayak"} {
		list = append(list, words.Word{Word: w, Category: "Technology", Clue: "A computer you can close."})
	}
	g, err := FromPackWithSeed(words.Pack{Name: "Fixture", GuessBudget: 8, List: list}, 7)
	if err != nil {
		t.Fatal(err)
	}
	g.Guess('Z')
	g.Hint()
	return g
}

func mustSnapshot(t *testing.T, g *Game) Snapshot {
	t.Helper()
	s, ok := g.Snapshot()
	if !ok {
		t.Fatal("the match is over, so there is no snapshot")
	}
	return s
}

// roundTrip is the same match, put through the trip a settings file gives it.
func roundTrip(t *testing.T, g *Game) *Game {
	t.Helper()
	resumed := Resume(mustSnapshot(t, g))
	if resumed == nil {
		t.Fatal("a snapshot straight off a live game was refused")
	}
	return resumed
}

// visible is everything a player can see about a game.
func visible(g *Game) string {
	return fmt.Sprintf("%v %q %d %s %s %d %d %d %d %d %d %v %s %q %q",
		g.Difficulty(), g.PackName(), g.PackWords(), g.Display(), g.Word(),
		g.WrongGuesses(), g.GuessBudget(), g.WordsWon(), g.WordsLost(),
		g.TotalWords(), g.WordNumber(), g.Result(), g.Guessed(),
		g.Category(), g.Clue())
}

func TestAMatchInFlightComesBackExactlyAsItWasLeft(t *testing.T) {
	g := matchInFlight(t)
	if got, want := visible(roundTrip(t, g)), visible(g); got != want {
		t.Errorf("resumed as\n%s\nwant\n%s", got, want)
	}
}

func TestAResumedWordKeepsItsCategoryAndItsClue(t *testing.T) {
	resumed := roundTrip(t, matchInFlight(t))
	if resumed.Category() != "Technology" || resumed.Clue() != "A computer you can close." {
		t.Errorf("category %q, clue %q", resumed.Category(), resumed.Clue())
	}
}

func TestAResumedMatchPlaysOnFromWhereItWas(t *testing.T) {
	resumed := roundTrip(t, matchInFlight(t))
	word := resumed.Word()
	winCurrentGame(resumed)
	if !resumed.IsWon() || resumed.WordsWon() != 1 {
		t.Errorf("won %v, %d won", resumed.IsWon(), resumed.WordsWon())
	}
	if !resumed.NextWord() {
		t.Fatal("no next word")
	}
	if resumed.Word() == word {
		t.Error("the pool did not move on to the next word")
	}
}

func TestAFinishedMatchIsNotWorthComingBackTo(t *testing.T) {
	g := gameWithWord(t, "LAPTOP")
	winCurrentGame(g)
	if !g.IsMatchOver() {
		t.Fatal("the match is not over")
	}
	if _, ok := g.Snapshot(); ok {
		t.Error("a finished match was saved")
	}
}

// The word is over but the match is not, and coming back to the next word
// of it beats coming back to a match that never happened.
func TestAWordThatHasJustBeenResolvedIsStillSaved(t *testing.T) {
	g := matchInFlight(t)
	loseCurrentGame(g)
	resumed := roundTrip(t, g)
	if resumed.Result() != Lost || resumed.WordsLost() != 1 || resumed.WordNumber() != 1 {
		t.Errorf("result %v, lost %d, word %d", resumed.Result(), resumed.WordsLost(), resumed.WordNumber())
	}
}

// The reason the result is stored rather than derived: this word has guesses
// in hand and letters still hidden, which is what a word in play looks like.
func TestAWordGivenUpOnDoesNotComeBackPlayable(t *testing.T) {
	g := matchInFlight(t)
	g.GiveUp()
	resumed := roundTrip(t, g)
	if !resumed.IsWordOver() || resumed.Result() != Lost {
		t.Errorf("over %v, result %v", resumed.IsWordOver(), resumed.Result())
	}
}

func TestAWordListOfYourOwnResumesWithItsOwnBudget(t *testing.T) {
	pack := words.Pack{GuessBudget: 9, List: []words.Word{words.Bare("Laptop"), words.Bare("Bagel")}}
	g, err := FromPackWithSeed(pack, 3)
	if err != nil {
		t.Fatal(err)
	}
	resumed := roundTrip(t, g)
	if resumed.Difficulty() != NoDifficulty || resumed.GuessBudget() != 9 {
		t.Errorf("difficulty %v, budget %d", resumed.Difficulty(), resumed.GuessBudget())
	}
}

func TestADifficultyGetsItsOwnBudgetBackWhateverTheFileSays(t *testing.T) {
	s := mustSnapshot(t, WithSeed(Insane, 4))
	// Insane's weight with Easy's slack, which the re-derivation refuses.
	s.GuessBudget = 10
	resumed := Resume(s)
	if resumed == nil {
		t.Fatal("only the budget was wrong")
	}
	if resumed.GuessBudget() != Insane.GuessBudget() {
		t.Errorf("budget %d", resumed.GuessBudget())
	}
}

func TestALoadedPackAskingForABudgetTheGallowsCannotDrawIsClamped(t *testing.T) {
	g := gameWithWord(t, "LAPTOP")
	for _, c := range []struct{ asked, want int }{{2, gallows.CoreParts}, {40, len(gallows.Parts)}} {
		s := mustSnapshot(t, g)
		s.GuessBudget = c.asked
		resumed := Resume(s)
		if resumed == nil {
			t.Fatalf("asked for %d: a budget is clamped, not refused", c.asked)
		}
		if resumed.GuessBudget() != c.want {
			t.Errorf("asked for %d, got %d", c.asked, resumed.GuessBudget())
		}
	}
}

func TestAWordWithNothingLeftToGuessIsNotResumed(t *testing.T) {
	g := gameWithWord(t, "LAPTOP")
	for _, w := range []string{"", "   ", "1234", "!!!"} {
		s := mustSnapshot(t, g)
		s.Current = words.Bare(w)
		if Resume(s) != nil {
			t.Errorf("%q was resumed", w)
		}
	}
}

func TestAResumedWordIsTrimmedAndUppercasedLikeAnyOther(t *testing.T) {
	s := mustSnapshot(t, gameWithWord(t, "LAPTOP"))
	s.Current = words.Bare("  laptop  ")
	resumed := Resume(s)
	if resumed == nil {
		t.Fatal("a playable word was refused")
	}
	if resumed.Word() != "LAPTOP" {
		t.Errorf("word %q", resumed.Word())
	}
}

func TestAResultTheRestOfTheStateCouldNotHaveProducedIsRefused(t *testing.T) {
	g := matchInFlight(t)
	g.Guess('Z')
	live := mustSnapshot(t, g)
	// A word on each side of the tally, so the tally check cannot refuse
	// these before the rule under test does.
	live.WordsWon, live.WordsLost = 1, 1

	notComplete := live
	notComplete.Result = Won

	spentInPlay := live
	spentInPlay.Wrong = live.GuessBudget
	spentInPlay.Result = InPlay

	completeLost := live
	completeLost.Guessed = ParseLetters(live.Current.Word)
	completeLost.Result = Lost

	for why, s := range map[string]Snapshot{
		"the word is not complete":  notComplete,
		"a spent budget is a loss":  spentInPlay,
		"a completed word is a win": completeLost,
	} {
		if Resume(s) != nil {
			t.Errorf("resumed although %s", why)
		}
	}
}

// Two words, not one: with an empty pool behind it a resolved word is refused
// by the finished-match check instead, and this test would pass without ever
// reaching the rule it is about.
func TestACompletedWordWithASpentBudgetIsNotResumed(t *testing.T) {
	live := mustSnapshot(t, mustFromWords(t, 9, "Laptop", "Bagel"))
	// A word on each side of the tally, so the tally check cannot refuse it
	// first either.
	live.WordsWon, live.WordsLost = 1, 1
	// Unreachable by playing, because the guess that empties the budget loses
	// the word first. Reachable by editing the file.
	for _, result := range []Result{Won, Lost, InPlay} {
		s := live
		s.Guessed = ParseLetters(live.Current.Word)
		s.Wrong = live.GuessBudget
		s.Result = result
		if Resume(s) != nil {
			t.Errorf("resumed as %v", result)
		}
	}
}

// A word that ended is on the tally, so the count for the way it ended
// cannot be nothing. Left in, the word falls out of WordNumber and out of the
// summary.
func TestAResolvedWordMissingFromTheTallyIsNotResumed(t *testing.T) {
	won := matchInFlight(t)
	winCurrentGame(won)
	lost := matchInFlight(t)
	lost.GiveUp()

	for zeroed, g := range map[string]*Game{"WordsWon": won, "WordsLost": lost} {
		s := mustSnapshot(t, g)
		s.WordsWon, s.WordsLost = 0, 0
		if Resume(s) != nil {
			t.Errorf("resumed with %s zero", zeroed)
		}
	}
}

func TestTheWordOnTheBoardIsAlwaysOneOfTheWordsPlayed(t *testing.T) {
	lost := matchInFlight(t)
	lost.GiveUp()
	for _, g := range []*Game{matchInFlight(t), lost} {
		resumed := roundTrip(t, g)
		if resumed.WordNumber() < 1 || resumed.WordNumber() > resumed.TotalWords() {
			t.Errorf("word %d of %d", resumed.WordNumber(), resumed.TotalWords())
		}
	}
}

// Given up on, so the snapshot is a plausible loss in every other respect,
// and only the budget check stands between it and a word carrying more wrong
// guesses than the gallows has stages.
func TestMoreWrongGuessesThanTheBudgetAllowsIsNotResumed(t *testing.T) {
	g := matchInFlight(t)
	g.GiveUp()
	s := mustSnapshot(t, g)
	s.Wrong = 99
	if Resume(s) != nil {
		t.Error("resumed with 99 wrong guesses")
	}
}

// The pool emptied out from under a word that is already over, so there is
// no next word and no summary to resume into.
func TestAResolvedWordWithNothingBehindItIsAFinishedMatchNotASavedOne(t *testing.T) {
	g := matchInFlight(t)
	loseCurrentGame(g)
	s := mustSnapshot(t, g)
	s.Remaining = nil
	if Resume(s) != nil {
		t.Error("a finished match was resumed")
	}
}

func TestTheMatchLengthIsRecountedRatherThanRestored(t *testing.T) {
	s := mustSnapshot(t, mustFromWords(t, 11, "Laptop", "Bagel", "Kayak", "Violin", "Anchor"))
	remaining := len(s.Remaining)
	s.WordsWon, s.WordsLost = 2, 1
	resumed := Resume(s)
	if resumed == nil {
		t.Fatal("three words played and the rest to come is an ordinary match")
	}
	// Three finished, one on the board, the rest still to deal.
	if resumed.TotalWords() != remaining+4 || resumed.WordNumber() != 4 {
		t.Errorf("word %d of %d, want 4 of %d", resumed.WordNumber(), resumed.TotalWords(), remaining+4)
	}
}

func TestAMatchLongerThanAMatchIsNotResumed(t *testing.T) {
	s := mustSnapshot(t, WithSeed(Easy, 12))
	s.WordsWon = MatchWords
	if Resume(s) != nil {
		t.Error("a match longer than a match was resumed")
	}
}

// The high bit stands in for anything a Letters can carry that is not A to Z.
func TestJunkInTheGuessedLettersIsDroppedRatherThanRefused(t *testing.T) {
	s := mustSnapshot(t, gameWithWord(t, "LAPTOP"))
	s.Guessed = ParseLetters("l4 z!") | 1<<31
	resumed := Resume(s)
	if resumed == nil {
		t.Fatal("the word is still playable")
	}
	if resumed.Guessed() != ParseLetters("LZ") {
		t.Errorf("guessed %q (%b), want LZ uppercased with everything unguessable dropped", resumed.Guessed(), uint32(resumed.Guessed()))
	}
	if resumed.Display() != "L_____" {
		t.Errorf("display %q", resumed.Display())
	}
}
