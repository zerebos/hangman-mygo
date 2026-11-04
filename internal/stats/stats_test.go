package stats

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// clean is a clean win on the classic budget. Nothing here reads the budget,
// so one fixed number keeps the tally tests comparable with each other.
const clean = game.DefaultGuessBudget

func won(s *Stats, d game.Difficulty) uint32 {
	return s.RecordWord(d, game.Won, clean)
}

func lost(s *Stats, d game.Difficulty) {
	s.RecordWord(d, game.Lost, clean)
}

// ------------------------------------------------------------ the formula

func TestTheFirstWordOfAStreakEarnsNoBonus(t *testing.T) {
	// (50 + 10 × 6) × 1 + 0
	if got := WordPoints(game.Easy, 6, 1); got != 110 {
		t.Errorf("got %d", got)
	}
}

// The budgets fall as the weights climb, and the weights win. If either table
// is retuned on its own, playing up could quietly stop paying.
func TestACleanWinPaysMoreTheHarderTheList(t *testing.T) {
	var got []uint32
	for _, d := range game.Difficulties {
		got = append(got, WordPoints(d, d.GuessBudget(), 1))
	}
	if !reflect.DeepEqual(got, []uint32{150, 260, 360, 440}) {
		t.Errorf("easy, medium, hard, insane: %v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Errorf("a clean win must be worth strictly more on every step up: %v", got)
		}
	}
}

func TestEveryRemainingGuessIsWorthTenBeforeTheWeight(t *testing.T) {
	for _, c := range []struct {
		d         game.Difficulty
		remaining int
		want      uint32
	}{
		{game.Easy, 0, 50},
		{game.Easy, 1, 60},
		{game.Medium, 1, 120},
	} {
		if got := WordPoints(c.d, c.remaining, 1); got != c.want {
			t.Errorf("%s with %d left: %d, want %d", c.d.Label(), c.remaining, got, c.want)
		}
	}
}

func TestACleanInsaneWinOnALongStreakIsFiveHundredAndForty(t *testing.T) {
	if got := WordPoints(game.Insane, 6, 5); got != 540 {
		t.Errorf("got %d", got)
	}
}

func TestTheStreakBonusStopsGrowingAfterFourSteps(t *testing.T) {
	capped := WordPoints(game.Easy, 6, 5)
	if capped != 110+StreakBonus*MaxStreakBonusSteps {
		t.Errorf("capped at %d", capped)
	}
	for _, streak := range []uint32{6, 20, math.MaxUint32} {
		if got := WordPoints(game.Easy, 6, streak); got != capped {
			t.Errorf("streak %d: %d, want %d", streak, got, capped)
		}
	}
}

func TestACustomWordListScoresAtWeightOne(t *testing.T) {
	if a, b := WordPoints(game.NoDifficulty, 6, 1), WordPoints(game.Easy, 6, 1); a != b {
		t.Errorf("custom %d, easy %d", a, b)
	}
}

func TestAbsurdInputsSaturateInsteadOfOverflowing(t *testing.T) {
	if got := WordPoints(game.Insane, math.MaxInt, 1); got != math.MaxUint32 {
		t.Errorf("max remaining: %d", got)
	}
	if got := WordPoints(game.NoDifficulty, 0, 0); got != BaseWordPoints {
		t.Errorf("streak 0: %d", got)
	}
	// Go's int is signed, so a negative count is the other absurd input.
	if got := WordPoints(game.Easy, -5, 1); got != BaseWordPoints {
		t.Errorf("negative remaining: %d", got)
	}
}

// -------------------------------------------------------------- the tally

func TestAFreshTallyIsEmpty(t *testing.T) {
	var s Stats
	if s.Points != 0 || s.WordsPlayed() != 0 || s.MatchesPlayed() != 0 || s.Streak != 0 || s.BestStreak != 0 {
		t.Errorf("stats %+v", s)
	}
	if s.WinRate() != 0 {
		t.Errorf("win rate %v", s.WinRate())
	}
	for _, d := range game.Difficulties {
		if b := s.ForDifficulty(d); b != (Bucket{}) {
			t.Errorf("%s bucket %+v", d.Label(), b)
		}
	}
}

func TestAWonWordScoresAndExtendsTheStreak(t *testing.T) {
	var s Stats
	points := won(&s, game.Medium)
	// (50 + 60) × 2, no bonus on the first word.
	if points != 220 || s.Points != 220 {
		t.Errorf("points %d, total %d", points, s.Points)
	}
	if s.WordsWon != 1 || s.Streak != 1 || s.BestStreak != 1 {
		t.Errorf("stats %+v", s)
	}
	if b := s.ForDifficulty(game.Medium); b.Points != 220 || b.WordsWon != 1 {
		t.Errorf("Medium bucket %+v", b)
	}
}

func TestTheSecondWordOfAStreakIsTheFirstToEarnABonus(t *testing.T) {
	var s Stats
	for i, want := range []uint32{110, 110 + StreakBonus, 110 + StreakBonus*2} {
		if got := won(&s, game.Easy); got != want {
			t.Errorf("word %d: %d, want %d", i+1, got, want)
		}
	}
	if s.Streak != 3 {
		t.Errorf("streak %d", s.Streak)
	}
}

func TestALostWordScoresNothingAndBreaksTheStreak(t *testing.T) {
	var s Stats
	won(&s, game.Easy)
	won(&s, game.Easy)
	if s.Streak != 2 {
		t.Fatalf("streak %d", s.Streak)
	}
	if got := s.RecordWord(game.Easy, game.Lost, 0); got != 0 {
		t.Errorf("a loss scored %d", got)
	}
	if s.Streak != 0 {
		t.Errorf("streak %d", s.Streak)
	}
	if s.BestStreak != 2 {
		t.Errorf("best streak %d, the best is a high-water mark", s.BestStreak)
	}
	if s.WordsLost != 1 || s.ForDifficulty(game.Easy).WordsLost != 1 {
		t.Errorf("lost %d, Easy lost %d", s.WordsLost, s.ForDifficulty(game.Easy).WordsLost)
	}
}

func TestTheStreakSurvivesADifficultyChange(t *testing.T) {
	var s Stats
	won(&s, game.Easy)
	won(&s, game.Insane)
	s.RecordMatch(game.Insane, game.MatchWon)
	won(&s, game.Hard)
	if s.Streak != 3 {
		t.Errorf("streak %d", s.Streak)
	}
}

func TestACustomWordListCountsTowardsTheTotalsButNoBucket(t *testing.T) {
	var s Stats
	points := s.RecordWord(game.NoDifficulty, game.Won, 6)
	s.RecordMatch(game.NoDifficulty, game.MatchWon)

	if points != 110 || s.Points != 110 || s.WordsWon != 1 || s.Streak != 1 || s.MatchesWon != 1 {
		t.Errorf("points %d, stats %+v", points, s)
	}
	for _, d := range game.Difficulties {
		if b := s.ForDifficulty(d); b != (Bucket{}) {
			t.Errorf("%s bucket %+v", d.Label(), b)
		}
	}
}

func TestMatchesAreCountedByOutcome(t *testing.T) {
	var s Stats
	s.RecordMatch(game.Hard, game.MatchWon)
	s.RecordMatch(game.Hard, game.MatchLost)
	s.RecordMatch(game.Hard, game.MatchTied)
	s.RecordMatch(game.Easy, game.MatchWon)

	if s.MatchesWon != 2 || s.MatchesLost != 1 || s.MatchesTied != 1 || s.MatchesPlayed() != 4 {
		t.Errorf("stats %+v", s)
	}
	hard := s.ForDifficulty(game.Hard)
	if hard.MatchesWon != 1 || hard.MatchesLost != 1 || hard.MatchesTied != 1 || hard.MatchesPlayed() != 3 {
		t.Errorf("Hard bucket %+v", hard)
	}
	if s.ForDifficulty(game.Easy).MatchesWon != 1 {
		t.Errorf("Easy bucket %+v", s.ForDifficulty(game.Easy))
	}
}

func TestTheWinRateIsTheShareOfWordsSolved(t *testing.T) {
	var s Stats
	won(&s, game.Easy)
	won(&s, game.Easy)
	won(&s, game.Easy)
	lost(&s, game.Easy)

	if s.WordsPlayed() != 4 || s.WinRate() != 0.75 || s.ForDifficulty(game.Easy).WinRate() != 0.75 {
		t.Errorf("played %d, rate %v, Easy rate %v", s.WordsPlayed(), s.WinRate(), s.ForDifficulty(game.Easy).WinRate())
	}
}

func TestResettingPutsEverythingBackToNothing(t *testing.T) {
	var s Stats
	won(&s, game.Insane)
	s.RecordMatch(game.Insane, game.MatchWon)
	s.Reset()
	if s != (Stats{}) {
		t.Errorf("stats %+v", s)
	}
}

// ------------------------------------------------------------ the session

func TestASessionAddsUpThePointsOfTheMatchOnScreen(t *testing.T) {
	session := NewSession(Stats{})
	first := session.RecordWord(game.Easy, game.Won, 6)
	second := session.RecordWord(game.Easy, game.Won, 6)
	if session.MatchPoints() != first+second || session.Stats().Points != uint64(first+second) {
		t.Errorf("match %d, lifetime %d, want %d", session.MatchPoints(), session.Stats().Points, first+second)
	}
}

func TestStartingAMatchZeroesTheScoreButNeverTheStreak(t *testing.T) {
	session := NewSession(Stats{})
	session.RecordWord(game.Easy, game.Won, 6)
	lifetime := session.Stats().Points

	session.StartMatch()

	if session.MatchPoints() != 0 {
		t.Errorf("match points %d", session.MatchPoints())
	}
	if session.Stats().Streak != 1 || session.Stats().BestStreak != 1 {
		t.Errorf("streak %d, best %d", session.Stats().Streak, session.Stats().BestStreak)
	}
	if session.Stats().Points != lifetime {
		t.Errorf("lifetime points %d, want %d kept", session.Stats().Points, lifetime)
	}
}

func TestASessionStartsFromTheStatsItIsHanded(t *testing.T) {
	var s Stats
	won(&s, game.Hard)
	session := NewSession(s)
	if *session.Stats() != s {
		t.Errorf("stats %+v, want %+v", *session.Stats(), s)
	}
	if session.MatchPoints() != 0 {
		t.Errorf("no match is in progress yet, but it has %d points", session.MatchPoints())
	}
}

func TestResettingASessionClearsTheMatchScoreToo(t *testing.T) {
	session := NewSession(Stats{})
	session.RecordWord(game.Easy, game.Won, 6)
	session.ResetStats()
	if session.MatchPoints() != 0 || *session.Stats() != (Stats{}) {
		t.Errorf("match %d, stats %+v", session.MatchPoints(), *session.Stats())
	}
}

// --------------------------------------------------------------- the file

func mustParse(t *testing.T, text string) Stats {
	t.Helper()
	var s Stats
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		t.Fatalf("parse %s: %v", text, err)
	}
	return s
}

func TestStatsSurviveARoundTrip(t *testing.T) {
	var s Stats
	won(&s, game.Easy)
	won(&s, game.Insane)
	lost(&s, game.Medium)
	s.RecordMatch(game.Insane, game.MatchTied)

	written, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if back := mustParse(t, string(written)); back != s {
		t.Errorf("read back %+v, want %+v", back, s)
	}
}

func TestTheBucketsAreStoredByName(t *testing.T) {
	var s Stats
	won(&s, game.Hard)
	written, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"best_streak":1`, `"Hard":{"points":330`, `"Easy":{"points":0`} {
		if !strings.Contains(string(written), want) {
			t.Errorf("%s is missing %s", written, want)
		}
	}
}

func TestAMissingBucketReadsAsAnEmptyOne(t *testing.T) {
	s := mustParse(t, `{"points": 40, "by_difficulty": {"Easy": {"points": 40}}}`)
	if s.Points != 40 || s.ForDifficulty(game.Easy).Points != 40 {
		t.Errorf("stats %+v", s)
	}
	if b := s.ForDifficulty(game.Insane); b != (Bucket{}) {
		t.Errorf("Insane bucket %+v", b)
	}
}

func TestAnUnknownBucketIsIgnoredRatherThanFailing(t *testing.T) {
	s := mustParse(t, `{"by_difficulty": {"Trivial": {"points": 9}, "hard": {"points": 3}}}`)
	// Known names still land, case-insensitively.
	if s.ForDifficulty(game.Hard).Points != 3 {
		t.Errorf("Hard bucket %+v", s.ForDifficulty(game.Hard))
	}
	for _, d := range []game.Difficulty{game.Easy, game.Medium, game.Insane} {
		if b := s.ForDifficulty(d); b != (Bucket{}) {
			t.Errorf("%s bucket %+v", d.Label(), b)
		}
	}
}

func TestABucketHoldingNonsenseReadsAsAnEmptyOne(t *testing.T) {
	s := mustParse(t, `{"points": 7, "by_difficulty": {"Easy": "lots"}}`)
	if s.Points != 7 {
		t.Errorf("points %d", s.Points)
	}
	if b := s.ForDifficulty(game.Easy); b != (Bucket{}) {
		t.Errorf("Easy bucket %+v", b)
	}
}

func TestAnEmptyObjectIsTheDefaultTally(t *testing.T) {
	if s := mustParse(t, "{}"); s != (Stats{}) {
		t.Errorf("stats %+v", s)
	}
}

// ----------------------------------------------------- hints cost guesses

// solved is a word with no repeated letters, so a hint can never finish it
// early.
const solved = "PLANET"

// solveWith solves solved from a one-word list, optionally spending a hint on
// the way, and reports what it scored and the streak it left. It drives a
// real game because the claim is that nothing here has to know hints exist.
func solveWith(t *testing.T, hint bool) (uint32, uint32) {
	t.Helper()
	g, err := game.FromWordsWithSeed(11, solved)
	if err != nil {
		t.Fatal(err)
	}
	session := NewSession(Stats{})
	if hint {
		if out := g.Hint(); out.Result != game.Revealed {
			t.Fatalf("a fresh word should have a hint to give, got %+v", out)
		}
	}
	for _, r := range solved {
		g.Guess(r)
	}
	if !g.IsWon() {
		t.Fatal("the word was not won")
	}
	points := session.RecordWord(g.Difficulty(), game.Won, g.RemainingGuesses())
	return points, session.Stats().Streak
}

// The only difference between the two is one unspent guess, which the
// formula already prices. There is no separate hint penalty.
func TestAHintedWinCostsOneGuessAndNothingElse(t *testing.T) {
	hinted, hintedStreak := solveWith(t, true)
	plain, plainStreak := solveWith(t, false)
	if plain-hinted != PointsPerRemainingGuess {
		t.Errorf("a hint cost %d points, want exactly one guess' worth", plain-hinted)
	}
	if want := WordPoints(game.NoDifficulty, game.DefaultGuessBudget-1, 1); hinted != want {
		t.Errorf("a hinted win scored %d, want %d", hinted, want)
	}
	if hintedStreak != plainStreak {
		t.Errorf("streaks %d and %d", hintedStreak, plainStreak)
	}
}

func TestAHintDoesNotBreakTheStreak(t *testing.T) {
	var s Stats
	won(&s, game.Hard)
	won(&s, game.Hard)
	if s.Streak != 2 {
		t.Fatalf("streak %d", s.Streak)
	}
	hinted := s.RecordWord(game.Hard, game.Won, game.DefaultGuessBudget-1)
	if s.Streak != 3 || s.BestStreak != 3 {
		t.Errorf("streak %d, best %d", s.Streak, s.BestStreak)
	}
	if want := WordPoints(game.Hard, game.DefaultGuessBudget-1, 3); hinted != want {
		t.Errorf("hinted %d, want %d", hinted, want)
	}
}

func TestAFreshTallyIsEmptyAndAnythingRecordedIsNot(t *testing.T) {
	var s Stats
	if !s.IsEmpty() {
		t.Error("a fresh tally is not empty")
	}
	s.RecordWord(game.Easy, game.Lost, 0)
	if s.IsEmpty() {
		t.Error("a recorded loss is empty")
	}
	s.Reset()
	if !s.IsEmpty() {
		t.Error("a reset tally is not empty")
	}
}

// Normal play never records a match with no words behind it, but the file is
// hand-editable and Reset would still clear this.
func TestAMatchOnItsOwnIsNotAnEmptyTally(t *testing.T) {
	var s Stats
	s.RecordMatch(game.Easy, game.MatchWon)
	if s.Points != 0 || s.WordsPlayed() != 0 || s.BestStreak != 0 {
		t.Errorf("stats %+v", s)
	}
	if s.IsEmpty() {
		t.Error("a match on its own is empty")
	}
}

func TestAResumedSessionKeepsTheMatchScoreAndTheTallyApart(t *testing.T) {
	played := NewSession(Stats{})
	played.RecordWord(game.Insane, game.Won, 5)
	played.RecordWord(game.Insane, game.Won, 4)

	resumed := ResumeSession(*played.Stats(), played.MatchPoints())

	if resumed.MatchPoints() != played.MatchPoints() || resumed.MatchPoints() == 0 {
		t.Errorf("match points %d, want %d", resumed.MatchPoints(), played.MatchPoints())
	}
	if *resumed.Stats() != *played.Stats() {
		t.Errorf("stats %+v, want %+v", *resumed.Stats(), *played.Stats())
	}
}
