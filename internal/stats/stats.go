// Package stats is points, streaks and the lifetime tally: the scoring model,
// with no UI in it.
//
// Package game knows the rules: whether a word was won and whether the match
// is over. It keeps no score beyond the per-match counts it needs to decide a
// match. Everything a player would call a score lives here.
//
// A solved word is worth
//
//	(BaseWordPoints + PointsPerRemainingGuess × guesses left) × difficulty weight
//	    + StreakBonus × min(streak − 1, MaxStreakBonusSteps)
//
// so a clean Insane win at the top of a streak is (50 + 10 × 6) × 4 + 25 × 4 =
// 540. A lost word is worth nothing.
//
// The streak counts consecutive words solved, and only failing a word ends
// it: it survives the end of a match, a difficulty switch, a custom word list
// and a restart. Giving up on a word fails it.
//
// A word list of your own has no difficulty, so it scores at weight 1 and
// counts towards the totals and the streak but towards none of the
// per-difficulty buckets.
package stats

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/zerebos/hangman-mygo/internal/game"
)

const (
	// BaseWordPoints is what solving a word is worth before anything else.
	BaseWordPoints = 50
	// PointsPerRemainingGuess is what each unspent wrong guess adds.
	PointsPerRemainingGuess = 10
	// StreakBonus is what each step of the streak bonus adds. The first word
	// of a streak earns none.
	StreakBonus = 25
	// MaxStreakBonusSteps caps the bonus, at a streak of five.
	MaxStreakBonusSteps = 4
)

// WordPoints is what one solved word scores. remainingGuesses is the budget
// left when it was finished, and streakAfter is the streak including this
// word, so 1 earns no bonus. Nothing here can overflow.
func WordPoints(d game.Difficulty, remainingGuesses int, streakAfter uint32) uint32 {
	remaining := uint32(min(max(remainingGuesses, 0), math.MaxInt32))
	base := addSat(BaseWordPoints, mulSat(PointsPerRemainingGuess, remaining))
	steps := min(subSat(streakAfter, 1), MaxStreakBonusSteps)
	return addSat(mulSat(base, uint32(d.Weight())), StreakBonus*steps)
}

// Bucket is the lifetime tally for one difficulty.
type Bucket struct {
	Points      uint64 `json:"points"`
	WordsWon    uint32 `json:"words_won"`
	WordsLost   uint32 `json:"words_lost"`
	MatchesWon  uint32 `json:"matches_won"`
	MatchesLost uint32 `json:"matches_lost"`
	MatchesTied uint32 `json:"matches_tied"`
}

// WordsPlayed is words won plus words lost.
func (b Bucket) WordsPlayed() uint32 { return addSat(b.WordsWon, b.WordsLost) }

// MatchesPlayed is matches won, lost and tied.
func (b Bucket) MatchesPlayed() uint32 {
	return addSat(addSat(b.MatchesWon, b.MatchesLost), b.MatchesTied)
}

// WinRate is the share of played words solved, from 0 to 1, and 0 when
// nothing was played.
func (b Bucket) WinRate() float32 { return winRate(b.WordsWon, b.WordsPlayed()) }

// Stats is everything the game remembers about how you have played, for ever.
// It is what package settings saves.
type Stats struct {
	Points      uint64
	WordsWon    uint32
	WordsLost   uint32
	Streak      uint32
	BestStreak  uint32
	MatchesWon  uint32
	MatchesLost uint32
	MatchesTied uint32
	// byDifficulty has one bucket per game.Difficulties entry, in order.
	byDifficulty [len(game.Difficulties)]Bucket
}

// ForDifficulty is the lifetime tally for one difficulty, or an empty one for
// NoDifficulty.
func (s *Stats) ForDifficulty(d game.Difficulty) Bucket {
	if b := s.bucket(d); b != nil {
		return *b
	}
	return Bucket{}
}

func (s *Stats) bucket(d game.Difficulty) *Bucket {
	for i, candidate := range game.Difficulties {
		if candidate == d {
			return &s.byDifficulty[i]
		}
	}
	return nil
}

// RecordWord records a finished word and returns what it scored. A win
// extends the streak first and then scores, so the word that takes the streak
// from 1 to 2 is the first to earn a bonus. A loss breaks the streak.
func (s *Stats) RecordWord(d game.Difficulty, result game.Result, remainingGuesses int) uint32 {
	bucket := s.bucket(d)
	if result != game.Won {
		s.Streak = 0
		s.WordsLost = addSat(s.WordsLost, 1)
		if bucket != nil {
			bucket.WordsLost = addSat(bucket.WordsLost, 1)
		}
		return 0
	}
	s.Streak = addSat(s.Streak, 1)
	s.BestStreak = max(s.BestStreak, s.Streak)
	s.WordsWon = addSat(s.WordsWon, 1)
	points := WordPoints(d, remainingGuesses, s.Streak)
	s.Points = addSat64(s.Points, uint64(points))
	if bucket != nil {
		bucket.WordsWon = addSat(bucket.WordsWon, 1)
		bucket.Points = addSat64(bucket.Points, uint64(points))
	}
	return points
}

// RecordMatch records a finished match. The streak spans matches, so it is
// not touched.
func (s *Stats) RecordMatch(d game.Difficulty, outcome game.MatchOutcome) {
	tally := func(won, lost, tied *uint32) {
		switch outcome {
		case game.MatchWon:
			*won = addSat(*won, 1)
		case game.MatchLost:
			*lost = addSat(*lost, 1)
		case game.MatchTied:
			*tied = addSat(*tied, 1)
		}
	}
	tally(&s.MatchesWon, &s.MatchesLost, &s.MatchesTied)
	if b := s.bucket(d); b != nil {
		tally(&b.MatchesWon, &b.MatchesLost, &b.MatchesTied)
	}
}

// MatchesPlayed is matches won, lost and tied.
func (s *Stats) MatchesPlayed() uint32 {
	return addSat(addSat(s.MatchesWon, s.MatchesLost), s.MatchesTied)
}

// WordsPlayed is words won plus words lost.
func (s *Stats) WordsPlayed() uint32 { return addSat(s.WordsWon, s.WordsLost) }

// WinRate is the share of played words solved, from 0 to 1.
func (s *Stats) WinRate() float32 { return winRate(s.WordsWon, s.WordsPlayed()) }

// Reset throws the lot away.
func (s *Stats) Reset() { *s = Stats{} }

// IsEmpty reports whether Reset would throw anything away. It compares every
// field, not only the ones a confirmation quotes, because the file is
// editable by hand and a match count can sit in it with nothing else.
func (s *Stats) IsEmpty() bool { return *s == Stats{} }

// statsJSON is the file's shape: the totals, plus the buckets keyed by the
// difficulty's name so a hand-edited file reads sensibly.
type statsJSON struct {
	Points       uint64                     `json:"points"`
	WordsWon     uint32                     `json:"words_won"`
	WordsLost    uint32                     `json:"words_lost"`
	Streak       uint32                     `json:"streak"`
	BestStreak   uint32                     `json:"best_streak"`
	MatchesWon   uint32                     `json:"matches_won"`
	MatchesLost  uint32                     `json:"matches_lost"`
	MatchesTied  uint32                     `json:"matches_tied"`
	ByDifficulty map[string]json.RawMessage `json:"by_difficulty"`
}

// MarshalJSON writes the buckets as an object keyed by difficulty name.
func (s Stats) MarshalJSON() ([]byte, error) {
	out := struct {
		statsJSON
		ByDifficulty map[string]Bucket `json:"by_difficulty"`
	}{
		statsJSON: statsJSON{
			Points: s.Points, WordsWon: s.WordsWon, WordsLost: s.WordsLost,
			Streak: s.Streak, BestStreak: s.BestStreak,
			MatchesWon: s.MatchesWon, MatchesLost: s.MatchesLost, MatchesTied: s.MatchesTied,
		},
		ByDifficulty: map[string]Bucket{},
	}
	for i, d := range game.Difficulties {
		out.ByDifficulty[d.Label()] = s.byDifficulty[i]
	}
	return json.Marshal(out)
}

// UnmarshalJSON reads what MarshalJSON writes, forgivingly: a difficulty this
// version does not know is dropped, and one whose bucket holds nonsense reads
// as empty rather than failing the whole value.
func (s *Stats) UnmarshalJSON(data []byte) error {
	var in statsJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	*s = Stats{
		Points: in.Points, WordsWon: in.WordsWon, WordsLost: in.WordsLost,
		Streak: in.Streak, BestStreak: in.BestStreak,
		MatchesWon: in.MatchesWon, MatchesLost: in.MatchesLost, MatchesTied: in.MatchesTied,
	}
	for name, raw := range in.ByDifficulty {
		for i, d := range game.Difficulties {
			if strings.EqualFold(d.Label(), name) {
				var b Bucket
				if json.Unmarshal(raw, &b) == nil {
					s.byDifficulty[i] = b
				}
			}
		}
	}
	return nil
}

// Session is the running score of the process: the lifetime Stats plus the
// points the match on screen has earned. It is what the view holds, so the
// view does no arithmetic of its own.
type Session struct {
	matchPoints uint32
	stats       Stats
}

// NewSession starts from the stats a previous run left behind, with no match
// score yet.
func NewSession(s Stats) *Session { return &Session{stats: s} }

// ResumeSession starts on a match already under way. The match score is the
// one thing about a match in flight that the game does not carry.
func ResumeSession(s Stats, matchPoints uint32) *Session {
	return &Session{matchPoints: matchPoints, stats: s}
}

// Stats is the lifetime tally.
func (s *Session) Stats() *Stats { return &s.stats }

// MatchPoints is what the match on screen has scored so far.
func (s *Session) MatchPoints() uint32 { return s.matchPoints }

// StartMatch begins a fresh match, and the match score goes back to zero. The
// streak is left alone. Dealing the next word of a match is not a new match.
func (s *Session) StartMatch() { s.matchPoints = 0 }

// RecordWord records a finished word, adds its points to the match, and
// returns them.
func (s *Session) RecordWord(d game.Difficulty, result game.Result, remainingGuesses int) uint32 {
	points := s.stats.RecordWord(d, result, remainingGuesses)
	s.matchPoints = addSat(s.matchPoints, points)
	return points
}

// RecordMatch records a finished match.
func (s *Session) RecordMatch(d game.Difficulty, outcome game.MatchOutcome) {
	s.stats.RecordMatch(d, outcome)
}

// ResetStats wipes the lifetime tally, and the match score with it.
func (s *Session) ResetStats() {
	s.stats.Reset()
	s.matchPoints = 0
}

func winRate(won, played uint32) float32 {
	if played == 0 {
		return 0
	}
	return float32(won) / float32(played)
}

func addSat(a, b uint32) uint32 {
	if a > math.MaxUint32-b {
		return math.MaxUint32
	}
	return a + b
}

func subSat(a, b uint32) uint32 {
	if a < b {
		return 0
	}
	return a - b
}

func mulSat(a, b uint32) uint32 {
	if a != 0 && b > math.MaxUint32/a {
		return math.MaxUint32
	}
	return a * b
}

func addSat64(a, b uint64) uint64 {
	if a > math.MaxUint64-b {
		return math.MaxUint64
	}
	return a + b
}
