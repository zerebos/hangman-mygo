// Package settings is the handful of choices that outlive a launch: the
// theme, the difficulty last picked, the lifetime stats and the match in
// flight, written as JSON to the platform's config directory.
//
// The window's size and place are not in here. mygo remembers them itself,
// under the window's StateKey.
//
// Nothing here may stop the game from starting. A missing file, an unreadable
// one, half a file or JSON of the wrong shape all fall back to the defaults,
// at worst with a line on stderr. A key holding nonsense costs only that key.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/stats"
	"github.com/zerebos/hangman-mygo/internal/words"
)

const (
	appDir   = "hangman-mygo"
	fileName = "settings.json"
)

// Theme is which palette the window starts in. The game is dark first, so
// the zero value is Dark.
type Theme int

const (
	Dark Theme = iota
	Light
)

func (t Theme) MarshalJSON() ([]byte, error) {
	if t == Light {
		return []byte(`"light"`), nil
	}
	return []byte(`"dark"`), nil
}

func (t *Theme) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	*t = Dark
	if strings.EqualFold(name, "light") {
		*t = Light
	}
	return nil
}

// Difficulty is a game.Difficulty stored by name, as "difficulty": "Insane".
// A name this version does not know reads as NoDifficulty, which is the same
// as never having picked one.
type Difficulty game.Difficulty

func (d Difficulty) MarshalJSON() ([]byte, error) {
	if label := game.Difficulty(d).Label(); label != "" {
		return json.Marshal(label)
	}
	return []byte("null"), nil
}

func (d *Difficulty) UnmarshalJSON(data []byte) error {
	*d = Difficulty(game.NoDifficulty)
	var name string
	if json.Unmarshal(data, &name) != nil {
		return nil
	}
	for _, candidate := range game.Difficulties {
		if strings.EqualFold(candidate.Label(), name) {
			*d = Difficulty(candidate)
		}
	}
	return nil
}

// SavedMatch is the file's shape for a game.Snapshot, plus the match score,
// which the snapshot does not carry.
//
// It is written on every guess rather than as the window closes, because a
// save that only happened on a clean close would hand a free reroll to anyone
// who killed the process mid-word. It checks nothing: what a plausible match
// is, is a rule, and game.Resume is where the rules are.
type SavedMatch struct {
	Word words.Word `json:"word"`
	// Guessed is every letter guessed, as one string such as "AEL".
	Guessed     string       `json:"guessed"`
	Wrong       int          `json:"wrong_guesses"`
	Result      string       `json:"result,omitempty"`
	Remaining   []words.Word `json:"remaining"`
	Difficulty  Difficulty   `json:"difficulty"`
	Pack        string       `json:"pack"`
	PackWords   int          `json:"pack_words"`
	GuessBudget int          `json:"guess_budget"`
	WordsWon    int          `json:"words_won"`
	WordsLost   int          `json:"words_lost"`
	MatchPoints uint32       `json:"match_points"`
}

// NewSavedMatch is the file's version of a snapshot and the match score that
// goes with it.
func NewSavedMatch(s game.Snapshot, matchPoints uint32) *SavedMatch {
	result := ""
	switch s.Result {
	case game.Won:
		result = "won"
	case game.Lost:
		result = "lost"
	}
	return &SavedMatch{
		Word:        s.Current,
		Guessed:     s.Guessed.String(),
		Wrong:       s.Wrong,
		Result:      result,
		Remaining:   s.Remaining,
		Difficulty:  Difficulty(s.Difficulty),
		Pack:        s.PackName,
		PackWords:   s.PackWords,
		GuessBudget: s.GuessBudget,
		WordsWon:    s.WordsWon,
		WordsLost:   s.WordsLost,
		MatchPoints: matchPoints,
	}
}

// Snapshot is the saved match back as a game.Snapshot, reshaped and nothing
// more. A result this version cannot read becomes one no game would accept,
// so Resume refuses it.
func (m *SavedMatch) Snapshot() game.Snapshot {
	result := game.InPlay
	switch strings.ToLower(m.Result) {
	case "":
	case "won":
		result = game.Won
	case "lost":
		result = game.Lost
	default:
		result = -1
	}
	return game.Snapshot{
		Difficulty:  game.Difficulty(m.Difficulty),
		PackName:    m.Pack,
		PackWords:   m.PackWords,
		GuessBudget: m.GuessBudget,
		Current:     m.Word,
		Guessed:     game.ParseLetters(m.Guessed),
		Wrong:       m.Wrong,
		Result:      result,
		Remaining:   m.Remaining,
		WordsWon:    m.WordsWon,
		WordsLost:   m.WordsLost,
	}
}

// Settings is everything the game remembers between launches.
type Settings struct {
	Theme Theme `json:"theme"`
	// Difficulty is the difficulty last chosen, or NoDifficulty until one is.
	Difficulty Difficulty `json:"difficulty"`
	// Stats is the lifetime score, streak and tally.
	Stats stats.Stats `json:"stats"`
	// InFlight is the match still being played when the game closed, or nil.
	InFlight *SavedMatch `json:"in_flight"`
}

// Path is where the file lives, or "" on a system with no config directory.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, appDir, fileName)
}

// Load reads the settings at path, or returns the defaults if there is any
// reason at all not to have them. A missing file is every first launch and is
// not worth a word.
func Load(path string) Settings {
	if path == "" {
		return Settings{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "hangman: could not read settings from %s: %v\n", path, err)
		}
		return Settings{}
	}
	return Parse(data)
}

// Parse reads a settings file, falling back to the defaults for anything that
// is not one. Each key is read on its own, so a stats value of the wrong shape
// costs the stats and not the theme.
func Parse(data []byte) Settings {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		fmt.Fprintf(os.Stderr, "hangman: ignoring unreadable settings (%v), using the defaults\n", err)
		return Settings{}
	}
	var s Settings
	if raw, ok := keys["theme"]; ok {
		_ = json.Unmarshal(raw, &s.Theme)
	}
	if raw, ok := keys["difficulty"]; ok {
		_ = json.Unmarshal(raw, &s.Difficulty)
	}
	if raw, ok := keys["stats"]; ok {
		var st stats.Stats
		if json.Unmarshal(raw, &st) == nil {
			s.Stats = st
		}
	}
	if raw, ok := keys["in_flight"]; ok {
		var m SavedMatch
		// A saved match without a word is no saved match.
		if json.Unmarshal(raw, &m) == nil && hasKey(raw, "word") {
			s.InFlight = &m
		}
	}
	return s
}

func hasKey(raw json.RawMessage, key string) bool {
	var keys map[string]json.RawMessage
	if json.Unmarshal(raw, &keys) != nil {
		return false
	}
	_, ok := keys[key]
	return ok
}

// Save writes the settings to path, reporting a failure on stderr and
// carrying on.
func (s Settings) Save(path string) {
	if path == "" {
		return
	}
	if err := s.writeTo(path); err != nil {
		fmt.Fprintf(os.Stderr, "hangman: could not save settings to %s: %v\n", path, err)
	}
}

// writeTo writes to a temporary file and renames it over the real one, so a
// save cut short leaves the previous settings rather than half a file.
func (s Settings) writeTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
