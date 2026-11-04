// Package game is the hangman rules, with no UI, no I/O and no globals.
//
// A game is one word: you guess letters until you reveal the word (a win) or
// run out of wrong guesses (a loss). A match is MatchWords games, drawn at
// random without replacement from a pack, so a match never repeats a word and
// a pack bigger than a match is not played out in one sitting. When the drawn
// words run out the match is over, and it is scored by comparing wins against
// losses.
package game

import (
	"math/rand/v2"
	"strings"

	"github.com/zerebos/hangman-mygo/assets"
	"github.com/zerebos/hangman-mygo/internal/gallows"
	"github.com/zerebos/hangman-mygo/internal/words"
)

// DefaultGuessBudget is the classic six wrong guesses, a whole hangman and no
// more. It is the budget of a word list with no difficulty behind it, because
// nothing about a file says how hard it is meant to be.
const DefaultGuessBudget = 6

// MatchWords is how many words one match is played over. A pack with fewer
// words than this is played in full.
const MatchWords = 10

// Difficulty is which bundled pack a match is played from. The zero value,
// NoDifficulty, is a word list of the player's own.
type Difficulty int

const (
	NoDifficulty Difficulty = iota
	Easy
	Medium
	Hard
	Insane
)

// Difficulties is every difficulty, in menu order.
var Difficulties = [...]Difficulty{Easy, Medium, Hard, Insane}

// Label is the name the original's Difficulty menu used, or "" for
// NoDifficulty.
func (d Difficulty) Label() string {
	switch d {
	case Easy:
		return "Easy"
	case Medium:
		return "Medium"
	case Hard:
		return "Hard"
	case Insane:
		return "Insane"
	}
	return ""
}

// Weight is how much the difficulty multiplies a solved word by: 1 to 4, and
// 1 for a word list of your own.
//
// The weight climbs as the budget falls, and the weight wins, so playing up
// always pays: a clean win is worth 150 on Easy, 260 on Medium, 360 on Hard
// and 440 on Insane.
func (d Difficulty) Weight() int {
	switch d {
	case Medium:
		return 2
	case Hard:
		return 3
	case Insane:
		return 4
	}
	return 1
}

// GuessBudget is how many wrong guesses the difficulty allows before the word
// is lost.
//
// The four numbers stay inside what the gallows can draw, gallows.CoreParts
// to len(gallows.Parts), so one wrong guess is always exactly one new body
// part. Insane keeps the original's six.
func (d Difficulty) GuessBudget() int {
	switch d {
	case Easy:
		return 10
	case Medium:
		return 8
	case Hard:
		return 7
	}
	return DefaultGuessBudget
}

// packFile is where the difficulty's pack lives in assets.Words.
func (d Difficulty) packFile() string {
	switch d {
	case Easy:
		return "words/easy.json"
	case Medium:
		return "words/med.json"
	case Hard:
		return "words/hard.json"
	case Insane:
		return "words/insane.json"
	}
	panic("game: NoDifficulty has no bundled pack")
}

// GuessResult is how a guess resolved.
type GuessResult int

const (
	// Correct means the letter is in the word.
	Correct GuessResult = iota
	// Wrong means the letter is not in the word, which costs a wrong guess.
	Wrong
	// Duplicate means the letter was already guessed. It costs nothing.
	Duplicate
	// Invalid means it was not an ASCII letter. It costs nothing.
	Invalid
	// Ignored means the game or the match was already over.
	Ignored
)

// Result is how a word ended. The zero value means it is still in play.
type Result int

const (
	InPlay Result = iota
	Won
	Lost
)

// MatchOutcome is how a match ended, by wins against losses. The zero value
// means the match is still running.
type MatchOutcome int

const (
	MatchRunning MatchOutcome = iota
	MatchWon
	MatchLost
	MatchTied
)

// GuessOutcome is everything one call to Guess changed.
//
// Word and Match are set only on the guess that ended the word or the match,
// so a UI can use them as "announce this now" signals. Unlike the original,
// the last word of a match reports both: the original skipped the word's own
// message and showed only the match summary.
type GuessOutcome struct {
	Result GuessResult
	Word   Result
	Match  MatchOutcome
}

// HintResult is how a hint request landed.
type HintResult int

const (
	// Revealed means a letter was revealed for the price of a wrong guess.
	Revealed HintResult = iota
	// NoGuessToSpare means one guess is left and a hint costs one. A hint
	// that spent the last guess would reveal a letter and lose the word in
	// the same breath, which is a trap rather than a choice.
	NoGuessToSpare
	// HintIgnored means the word or the match was over, or nothing was left
	// hidden.
	HintIgnored
)

// HintOutcome is everything one call to Hint changed. A hint can only end a
// word by winning it.
type HintOutcome struct {
	Result HintResult
	// Letter is the letter revealed, when Result is Revealed.
	Letter rune
	Word   Result
	Match  MatchOutcome
}

// Cell is one character of the current word, ready to be laid out.
type Cell struct {
	// Value is the character, uppercase for letters.
	Value rune
	// Guessable is false for anything but A to Z: spaces and slashes show
	// for free and the win check ignores them.
	Guessable bool
	// Revealed is whether the character is shown rather than hidden.
	Revealed bool
}

// Display is the character to draw: the value, or '_' while it is hidden.
func (c Cell) Display() rune {
	if c.Revealed {
		return c.Value
	}
	return '_'
}

// Letters is a set of the letters A to Z.
type Letters uint32

// allLetters is every letter from A to Z.
const allLetters Letters = 1<<26 - 1

func letterBit(r rune) Letters {
	return 1 << uint(r-'A')
}

// Has reports whether r, an uppercase letter, is in the set.
func (l Letters) Has(r rune) bool {
	return 'A' <= r && r <= 'Z' && l&letterBit(r) != 0
}

// With is the set with r added. Anything but a letter is left out, and a
// lowercase letter goes in as its uppercase.
func (l Letters) With(r rune) Letters {
	if 'a' <= r && r <= 'z' {
		r -= 'a' - 'A'
	}
	if r < 'A' || r > 'Z' {
		return l
	}
	return l | letterBit(r)
}

// Len is how many letters are in the set.
func (l Letters) Len() int {
	n := 0
	for ; l != 0; l &= l - 1 {
		n++
	}
	return n
}

// String is the letters in alphabetical order, as "AEL".
func (l Letters) String() string {
	var b strings.Builder
	for r := 'A'; r <= 'Z'; r++ {
		if l.Has(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ParseLetters reads a set back out of a string, dropping anything that is
// not a letter.
func ParseLetters(s string) Letters {
	var l Letters
	for _, r := range s {
		l = l.With(r)
	}
	return l
}

// Snapshot is a match in flight, in the detail it takes to pick it up again.
//
// It holds only what cannot be worked out from anything else. What a Game
// derives (the cells, the letters left, which word of the match this is, the
// match's length) is left out, because derived state written down twice can
// come back disagreeing with itself. The words are stored by value rather than
// as positions in a pack, so a pack edited or deleted between launches costs
// nothing.
type Snapshot struct {
	Difficulty  Difficulty
	PackName    string
	PackWords   int
	GuessBudget int
	Current     words.Word
	Guessed     Letters
	Wrong       int
	// Result is stored rather than derived because a word given up on is
	// lost with guesses in hand and letters still hidden, which looks exactly
	// like a word still in play.
	Result    Result
	Remaining []words.Word
	WordsWon  int
	WordsLost int
}

// budgetFor is the guess budget a match is played with.
//
// A difficulty's own budget is the rule for it, and a pack reached through a
// difficulty does not get to argue. A pack the player loaded may state one,
// which is pulled into what the gallows can draw, and otherwise gets
// DefaultGuessBudget.
func budgetFor(d Difficulty, packBudget int) int {
	if d != NoDifficulty {
		return d.GuessBudget()
	}
	if packBudget == 0 {
		return DefaultGuessBudget
	}
	return min(max(packBudget, gallows.CoreParts), len(gallows.Parts))
}

// drawMatch draws the words one match is played over, at most MatchWords of
// them, without replacement.
func drawMatch(list []words.Word, rng *rand.Rand) []words.Word {
	list = append([]words.Word(nil), list...)
	take := min(len(list), MatchWords)
	drawn := make([]words.Word, 0, take)
	for range take {
		i := rng.IntN(len(list))
		drawn = append(drawn, list[i])
		list[i] = list[len(list)-1]
		list = list[:len(list)-1]
	}
	return drawn
}

// Game is a hangman match in progress. The first word is dealt as it is made,
// so a new Game is playable at once.
type Game struct {
	difficulty Difficulty
	// remaining is the words not yet dealt this match.
	remaining  []words.Word
	totalWords int
	packWords  int
	packName   string
	current    words.Word
	guessed    Letters
	wrong      int
	budget     int
	result     Result
	// wordsWon and wordsLost are per match and exist so finishMatch can
	// decide the outcome. Points and streaks are package stats' business.
	wordsWon  int
	wordsLost int
	outcome   MatchOutcome
	rng       *rand.Rand
}

// ErrEmptyWordList is returned for a word list with nothing playable in it.
var ErrEmptyWordList = emptyWordList{}

type emptyWordList struct{}

func (emptyWordList) Error() string { return "the word list contains no playable words" }

func newRand() *rand.Rand {
	return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
}

func seeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

// New starts a match on d, seeded at random.
func New(d Difficulty) *Game {
	return start(d, BundledPack(d), newRand())
}

// WithSeed starts a match on d with a fixed seed: the same seed deals the same
// words in the same order and hints the same letters.
func WithSeed(d Difficulty, seed uint64) *Game {
	return start(d, BundledPack(d), seeded(seed))
}

// FromPack starts a match on a pack the player opened. Its words keep their
// categories and clues, and it may state its own guess budget.
func FromPack(pack words.Pack) (*Game, error) {
	return fromPack(pack, newRand())
}

// FromPackWithSeed is FromPack with a fixed seed.
func FromPackWithSeed(pack words.Pack, seed uint64) (*Game, error) {
	return fromPack(pack, seeded(seed))
}

// FromWords starts a match on a plain list of words.
func FromWords(list ...string) (*Game, error) {
	return fromPack(words.FromWords(list...), newRand())
}

// FromWordsWithSeed is FromWords with a fixed seed.
func FromWordsWithSeed(seed uint64, list ...string) (*Game, error) {
	return fromPack(words.FromWords(list...), seeded(seed))
}

func fromPack(pack words.Pack, rng *rand.Rand) (*Game, error) {
	list := pack.Words()
	if len(list) == 0 {
		return nil, ErrEmptyWordList
	}
	return startWith(NoDifficulty, budgetFor(NoDifficulty, pack.GuessBudget), pack.DisplayName(), list, rng), nil
}

func start(d Difficulty, pack words.Pack, rng *rand.Rand) *Game {
	return startWith(d, budgetFor(d, pack.GuessBudget), pack.DisplayName(), pack.Words(), rng)
}

func startWith(d Difficulty, budget int, name string, list []words.Word, rng *rand.Rand) *Game {
	g := &Game{budget: budget, rng: rng}
	g.resetWith(d, budget, name, list)
	return g
}

// Resume picks a match back up from a Snapshot, or returns nil for one that
// does not describe a match anyone could be in the middle of.
//
// A snapshot comes out of a file the player is invited to edit, so it is
// checked the way a pack off disk is, and a refusal costs a fresh deal rather
// than a crash:
//
//   - The word and the pool go through words.Sanitize. A current word with
//     nothing left to guess is refused.
//   - The budget is re-derived through budgetFor rather than restored, so a
//     hand edit cannot buy Easy's ten guesses at Insane's weight.
//   - Result has to be one the rest of the state could have produced: a
//     spent budget is a loss, a completed word is a win, and anything else is
//     in play or given up on.
//   - A resolved word has to be on the per-match tally, because the game
//     counts a word as it sets the result.
//   - A resolved word with an empty pool behind it is a finished match,
//     which Snapshot never writes.
//   - The match cannot be longer than MatchWords.
//
// The random source is fresh. Word order is already decided by the pool, so
// a seed would only choose which letter a hint reveals.
func Resume(s Snapshot) *Game {
	return resume(s, newRand())
}

// ResumeWithSeed is Resume with a fixed seed.
func ResumeWithSeed(s Snapshot, seed uint64) *Game {
	return resume(s, seeded(seed))
}

func resume(s Snapshot, rng *rand.Rand) *Game {
	current := words.Sanitize([]words.Word{s.Current})
	if len(current) == 0 {
		return nil
	}
	remaining := words.Sanitize(s.Remaining)
	if s.Difficulty < NoDifficulty || s.Difficulty > Insane {
		return nil
	}
	budget := budgetFor(s.Difficulty, s.GuessBudget)
	if s.Wrong < 0 || s.Wrong > budget || s.WordsWon < 0 || s.WordsLost < 0 {
		return nil
	}

	// Only A to Z survive, so a stray bit cannot make an untouched word look
	// like one in progress.
	guessed := s.Guessed & allLetters
	complete := wordComplete(current[0].Word, guessed)
	spent := s.Wrong >= budget
	// A spent budget is asked about first, and it rules out a completed word
	// as well as any result but a loss: the guess that empties the budget
	// loses the word, and a hint refuses at one guess left so it can never be
	// that guess. So nothing can complete a word after that.
	var plausible bool
	switch {
	case spent:
		plausible = !complete && s.Result == Lost
	case complete:
		plausible = s.Result == Won
	default:
		plausible = s.Result == InPlay || s.Result == Lost
	}
	if !plausible {
		return nil
	}

	// A resolved word is already on the tally. Without this check the word on
	// the board falls out of the count and the panel reads "word 0 of 3".
	if (s.Result == Won && s.WordsWon == 0) || (s.Result == Lost && s.WordsLost == 0) {
		return nil
	}
	if s.Result != InPlay && len(remaining) == 0 {
		return nil
	}

	// The match's length, recounted rather than restored. A word still in
	// play is not on the tally yet, so it is counted on top. The counts come
	// out of a file, so each is checked before the sum can overflow.
	if s.WordsWon > MatchWords || s.WordsLost > MatchWords || len(remaining) > MatchWords {
		return nil
	}
	total := s.WordsWon + s.WordsLost + len(remaining)
	if s.Result == InPlay {
		total++
	}
	if total > MatchWords {
		return nil
	}

	return &Game{
		difficulty: s.Difficulty,
		// A pack cannot have held fewer words than the match drew from it; a
		// file saying so is only ever wrong about a sentence in a toast.
		packWords:  max(s.PackWords, total),
		packName:   strings.TrimSpace(s.PackName),
		remaining:  remaining,
		totalWords: total,
		current:    current[0],
		guessed:    guessed,
		wrong:      s.Wrong,
		budget:     budget,
		result:     s.Result,
		wordsWon:   s.WordsWon,
		wordsLost:  s.WordsLost,
		rng:        rng,
	}
}

// bundledFS holds the four packs the difficulties play from.
var bundledFS = assets.Words

// BundledPack is the pack a difficulty plays from. It panics for
// NoDifficulty, and for a bundled pack that does not parse, which a test rules
// out.
func BundledPack(d Difficulty) words.Pack {
	data, err := bundledFS.ReadFile(d.packFile())
	if err != nil {
		panic(err)
	}
	return words.MustParseBundled(data)
}

// dealWord takes a random word out of the pool, and reports false when the
// pool was empty.
func (g *Game) dealWord() bool {
	if len(g.remaining) == 0 {
		return false
	}
	i := g.rng.IntN(len(g.remaining))
	g.current = g.remaining[i]
	last := len(g.remaining) - 1
	g.remaining[i] = g.remaining[last]
	g.remaining = g.remaining[:last]
	g.guessed = 0
	g.wrong = 0
	g.result = InPlay
	return true
}

// ------------------------------------------------------------------ actions

// Guess guesses a letter. Case does not matter. Non-letters, repeats and
// guesses after the word is over cost nothing.
func (g *Game) Guess(letter rune) GuessOutcome {
	if g.IsWordOver() {
		return GuessOutcome{Result: Ignored}
	}
	if !words.IsLetter(letter) {
		return GuessOutcome{Result: Invalid}
	}
	if 'a' <= letter && letter <= 'z' {
		letter -= 'a' - 'A'
	}
	if g.guessed.Has(letter) {
		return GuessOutcome{Result: Duplicate}
	}
	g.guessed = g.guessed.With(letter)

	if strings.ContainsRune(g.current.Word, letter) {
		if g.isWordComplete() {
			return g.endWord(Won, Correct)
		}
		return GuessOutcome{Result: Correct}
	}
	g.wrong++
	if g.wrong >= g.budget {
		return g.endWord(Lost, Wrong)
	}
	return GuessOutcome{Result: Wrong}
}

// Hint reveals one hidden letter at the price of one wrong guess.
//
// The letter goes into the guessed set exactly as a correct guess would, and
// the wrong-guess count goes up by one, so the word, the keyboard, the
// gallows and the score all charge for it without knowing hints exist. The
// letter is drawn from the game's own random source, so a seeded game hints
// reproducibly.
func (g *Game) Hint() HintOutcome {
	if g.IsWordOver() {
		return HintOutcome{Result: HintIgnored}
	}
	// One guess left is refused, not zero: see NoGuessToSpare.
	if g.RemainingGuesses() <= 1 {
		return HintOutcome{Result: NoGuessToSpare}
	}
	hidden := g.hiddenLetters()
	if len(hidden) == 0 {
		return HintOutcome{Result: HintIgnored}
	}

	letter := hidden[g.rng.IntN(len(hidden))]
	g.guessed = g.guessed.With(letter)
	g.wrong++

	// Only the win is checked: at least two guesses were left before the
	// charge, so the budget cannot run out here.
	out := HintOutcome{Result: Revealed, Letter: letter}
	if g.isWordComplete() {
		ended := g.endWord(Won, Correct)
		out.Word, out.Match = ended.Word, ended.Match
	}
	return out
}

// GiveUp loses the current word at once, and returns the match's outcome if
// it was the last word. It does nothing on a word that is already over.
func (g *Game) GiveUp() MatchOutcome {
	if g.IsWordOver() {
		return MatchRunning
	}
	return g.endWord(Lost, Ignored).Match
}

func (g *Game) endWord(result Result, guess GuessResult) GuessOutcome {
	g.result = result
	if result == Won {
		g.wordsWon++
	} else {
		g.wordsLost++
	}
	out := GuessOutcome{Result: guess, Word: result}
	if len(g.remaining) == 0 {
		g.finishMatch()
		out.Match = g.outcome
	}
	return out
}

func (g *Game) finishMatch() {
	switch {
	case g.wordsWon > g.wordsLost:
		g.outcome = MatchWon
	case g.wordsWon < g.wordsLost:
		g.outcome = MatchLost
	default:
		g.outcome = MatchTied
	}
}

// NextWord deals the next word of the match. It reports false, and changes
// nothing, when the match is over or the current word is still in play.
func (g *Game) NextWord() bool {
	if g.IsMatchOver() || !g.IsWordOver() {
		return false
	}
	return g.dealWord()
}

// SetDifficulty starts a fresh match on d.
//
// It reports false, and changes nothing, when d is the difficulty already in
// play and the match is still running, so a stray click on the selected pill
// cannot cost the word in hand. Once the match is over that click is the only
// way to replay the list, so it restarts.
func (g *Game) SetDifficulty(d Difficulty) bool {
	if !g.WouldSwitchTo(d) {
		return false
	}
	pack := BundledPack(d)
	g.resetWith(d, budgetFor(d, pack.GuessBudget), pack.DisplayName(), pack.Words())
	return true
}

// WouldSwitchTo reports whether SetDifficulty would do anything for d.
func (g *Game) WouldSwitchTo(d Difficulty) bool {
	return g.difficulty != d || g.IsMatchOver()
}

// HasWordToLose reports whether walking away from the word now would cost
// it: true once a guess or a hint has landed on a word still in play. Picking
// a difficulty before playing a letter is free.
func (g *Game) HasWordToLose() bool {
	return !g.IsWordOver() && g.guessed != 0
}

// SetPack starts a fresh match on a pack the player opened. A pack with
// nothing playable is refused and the match is left as it was.
func (g *Game) SetPack(pack words.Pack) error {
	list := pack.Words()
	if len(list) == 0 {
		return ErrEmptyWordList
	}
	g.resetWith(NoDifficulty, budgetFor(NoDifficulty, pack.GuessBudget), pack.DisplayName(), list)
	return nil
}

// SetWordList is SetPack for a plain list of words.
func (g *Game) SetWordList(list ...string) error {
	return g.SetPack(words.FromWords(list...))
}

func (g *Game) resetWith(d Difficulty, budget int, name string, list []words.Word) {
	g.packWords = len(list)
	g.packName = name
	drawn := drawMatch(list, g.rng)
	g.difficulty = d
	g.budget = budget
	g.totalWords = len(drawn)
	g.remaining = drawn
	g.wordsWon = 0
	g.wordsLost = 0
	g.outcome = MatchRunning
	if !g.dealWord() {
		g.finishMatch()
	}
}

// ------------------------------------------------------------------ queries

// Word is the current word, uppercased.
func (g *Game) Word() string { return g.current.Word }

// Category is what kind of thing the current word is, or "".
func (g *Game) Category() string { return g.current.Category }

// Clue is a sentence about what the current word means, or "". Reading it
// costs nothing, so the game does not track whether it was read.
func (g *Game) Clue() string { return g.current.Clue }

// Cells is the current word as cells, one per character. Once the word is
// over every cell is revealed.
func (g *Game) Cells() []Cell {
	over := g.IsWordOver()
	var cells []Cell
	for _, r := range g.current.Word {
		guessable := words.IsLetter(r)
		cells = append(cells, Cell{
			Value:     r,
			Guessable: guessable,
			Revealed:  over || !guessable || g.guessed.Has(r),
		})
	}
	return cells
}

// Display is the word as revealed letters and '_', as "A__LES_UCE".
func (g *Game) Display() string {
	var b strings.Builder
	for _, c := range g.Cells() {
		b.WriteRune(c.Display())
	}
	return b.String()
}

// Guessed is every letter guessed against this word, hints included.
func (g *Game) Guessed() Letters { return g.guessed }

// AvailableLetters is the letters not guessed yet, in order.
func (g *Game) AvailableLetters() []rune {
	var out []rune
	for r := 'A'; r <= 'Z'; r++ {
		if !g.guessed.Has(r) {
			out = append(out, r)
		}
	}
	return out
}

// WrongGuesses is how many wrong guesses this word has cost, hints included.
func (g *Game) WrongGuesses() int { return g.wrong }

// GuessBudget is how many wrong guesses this match allows.
func (g *Game) GuessBudget() int { return g.budget }

// RemainingGuesses is how many wrong guesses are still affordable.
func (g *Game) RemainingGuesses() int { return max(g.budget-g.wrong, 0) }

// CanHint reports whether Hint would reveal something. A hint is unavailable
// one guess before the last, not on it.
func (g *Game) CanHint() bool {
	return !g.IsWordOver() && g.RemainingGuesses() > 1 && len(g.hiddenLetters()) > 0
}

// IsWordOver reports whether the current word has been won or lost.
func (g *Game) IsWordOver() bool { return g.result != InPlay }

// Result is how the current word ended, or InPlay.
func (g *Game) Result() Result { return g.result }

// IsWon reports whether the current word was won.
func (g *Game) IsWon() bool { return g.result == Won }

// WordsWon is the words won so far this match.
func (g *Game) WordsWon() int { return g.wordsWon }

// WordsLost is the words lost so far this match.
func (g *Game) WordsLost() int { return g.wordsLost }

// Snapshot is the match in flight, or false when the match is over and there
// is nothing to come back to. Every other state is worth saving, including a
// word just resolved and one not guessed at yet.
func (g *Game) Snapshot() (Snapshot, bool) {
	if g.IsMatchOver() {
		return Snapshot{}, false
	}
	return Snapshot{
		Difficulty:  g.difficulty,
		PackName:    g.packName,
		PackWords:   g.packWords,
		GuessBudget: g.budget,
		Current:     g.current,
		Guessed:     g.guessed,
		Wrong:       g.wrong,
		Result:      g.result,
		Remaining:   append([]words.Word(nil), g.remaining...),
		WordsWon:    g.wordsWon,
		WordsLost:   g.wordsLost,
	}, true
}

// Difficulty is the difficulty being played, or NoDifficulty for a word list
// of the player's.
func (g *Game) Difficulty() Difficulty { return g.difficulty }

// IsMatchOver reports whether every word of the match has been played.
func (g *Game) IsMatchOver() bool { return g.outcome != MatchRunning }

// MatchOutcome is how the match ended, or MatchRunning.
func (g *Game) MatchOutcome() MatchOutcome { return g.outcome }

// TotalWords is how many words the match started with.
func (g *Game) TotalWords() int { return g.totalWords }

// PackWords is how many playable words the pack held, which is more than
// TotalWords whenever the pack is bigger than a match.
func (g *Game) PackWords() int { return g.packWords }

// PackName is what the pack called itself, or "".
func (g *Game) PackName() string { return g.packName }

// WordNumber is which word of the match is on the board, from 1.
func (g *Game) WordNumber() int { return g.totalWords - len(g.remaining) }

// hiddenLetters is the distinct letters of the word still hidden, sorted, so
// which letter a seeded game hints depends on the seed alone.
func (g *Game) hiddenLetters() []rune {
	var hidden Letters
	for _, r := range g.current.Word {
		if words.IsLetter(r) && !g.guessed.Has(r) {
			hidden = hidden.With(r)
		}
	}
	return []rune(hidden.String())
}

func (g *Game) isWordComplete() bool {
	return wordComplete(g.current.Word, g.guessed)
}

// wordComplete reports whether every letter of word is in guessed. It is free
// rather than a method because Resume asks it of a word that is not a Game
// yet, and asking one question two ways is how two checks come to disagree.
func wordComplete(word string, guessed Letters) bool {
	for _, r := range word {
		if words.IsLetter(r) && !guessed.Has(r) {
			return false
		}
	}
	return true
}
