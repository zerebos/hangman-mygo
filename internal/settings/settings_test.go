package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/stats"
	"github.com/zerebos/hangman-mygo/internal/words"
)

// savedMatch is a match in flight with every field saying something, so a
// round trip that drops one is a round trip that fails.
func savedMatch() *SavedMatch {
	return &SavedMatch{
		Word:        words.Word{Word: "LAPTOP", Category: "Technology", Clue: "A computer you can close."},
		Guessed:     "AOPT",
		Wrong:       1,
		Remaining:   []words.Word{words.Bare("BAGEL"), words.Bare("KAYAK")},
		Difficulty:  Difficulty(game.Medium),
		Pack:        "Medium",
		PackWords:   30,
		GuessBudget: 8,
		WordsWon:    2,
		WordsLost:   1,
		MatchPoints: 520,
	}
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func marshalIndent(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// -------------------------------------------------------------- the file

func TestDefaultsAreADarkWindowWithNothingRestored(t *testing.T) {
	var s Settings
	if s.Theme != Dark || game.Difficulty(s.Difficulty) != game.NoDifficulty || s.Stats != (stats.Stats{}) || s.InFlight != nil {
		t.Errorf("defaults %+v", s)
	}
}

func TestSettingsSurviveARoundTrip(t *testing.T) {
	s := Settings{
		Theme:      Light,
		Difficulty: Difficulty(game.Insane),
		InFlight:   savedMatch(),
	}
	if back := Parse(marshalIndent(t, s)); !reflect.DeepEqual(back, s) {
		t.Errorf("read back %+v, want %+v", back, s)
	}
}

func TestSettingsSurviveATripThroughTheDisk(t *testing.T) {
	// A directory that does not exist yet, as on a first launch.
	path := filepath.Join(t.TempDir(), "hangman-mygo", "settings.json")
	s := Settings{Theme: Light, Difficulty: Difficulty(game.Hard), InFlight: savedMatch()}
	s.Stats.RecordWord(game.Hard, game.Won, 7)

	s.Save(path)

	if back := Load(path); !reflect.DeepEqual(back, s) {
		t.Errorf("loaded %+v, want %+v", back, s)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("the temporary file was left behind: %v", err)
	}
}

func TestAMissingFileIsTheDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if s := Load(path); !reflect.DeepEqual(s, Settings{}) {
		t.Errorf("loaded %+v", s)
	}
	if s := Load(""); !reflect.DeepEqual(s, Settings{}) {
		t.Errorf("loaded %+v with no path", s)
	}
}

func TestTheFileStoresReadableNames(t *testing.T) {
	text := marshal(t, Settings{Theme: Light, Difficulty: Difficulty(game.Medium)})
	for _, want := range []string{`"theme":"light"`, `"difficulty":"Medium"`} {
		if !strings.Contains(text, want) {
			t.Errorf("%s is missing %s", text, want)
		}
	}
}

func TestCorruptContentsFallBackToTheDefaults(t *testing.T) {
	for _, text := range []string{"", "{", "not json at all", `{"theme": 7}`, "[]"} {
		if s := Parse([]byte(text)); !reflect.DeepEqual(s, Settings{}) {
			t.Errorf("%q read as %+v", text, s)
		}
	}
}

func TestMissingFieldsTakeTheirDefaults(t *testing.T) {
	s := Parse([]byte(`{"theme": "light"}`))
	if s.Theme != Light || game.Difficulty(s.Difficulty) != game.NoDifficulty {
		t.Errorf("settings %+v", s)
	}
}

func TestUnknownFieldsAreIgnored(t *testing.T) {
	if s := Parse([]byte(`{"theme": "light", "hints": true}`)); s.Theme != Light {
		t.Errorf("theme %v", s.Theme)
	}
}

func TestAnUnknownDifficultyReadsAsNone(t *testing.T) {
	if d := Parse([]byte(`{"difficulty": "Trivial"}`)).Difficulty; game.Difficulty(d) != game.NoDifficulty {
		t.Errorf("difficulty %v", d)
	}
}

func TestDifficultyNamesAreMatchedCaseInsensitively(t *testing.T) {
	if d := Parse([]byte(`{"difficulty": "hard"}`)).Difficulty; game.Difficulty(d) != game.Hard {
		t.Errorf("difficulty %v", d)
	}
}

func TestEveryDifficultyRoundTripsByName(t *testing.T) {
	for _, d := range game.Difficulties {
		text := marshal(t, Settings{Difficulty: Difficulty(d)})
		if back := Parse([]byte(text)).Difficulty; game.Difficulty(back) != d {
			t.Errorf("%s read back as %v", d.Label(), back)
		}
	}
}

// ------------------------------------------------------------- the stats

// playedStats is a tally with something in every part of it, so a round
// trip has something to lose.
func playedStats() stats.Stats {
	var s stats.Stats
	s.RecordWord(game.Medium, game.Lost, 0)
	// A clean Insane win, a scrappy Easy one and a custom list, three in a
	// row, so the streak and its bonus are in there too.
	s.RecordWord(game.Insane, game.Won, 6)
	s.RecordWord(game.Easy, game.Won, 2)
	s.RecordWord(game.NoDifficulty, game.Won, 4)
	s.RecordMatch(game.Insane, game.MatchWon)
	return s
}

func TestAFileFromBeforeTheStatsExistedReadsAsAnEmptyTally(t *testing.T) {
	s := Parse([]byte(`{"theme": "light", "difficulty": "Hard"}`))
	if s.Theme != Light || game.Difficulty(s.Difficulty) != game.Hard || s.Stats != (stats.Stats{}) {
		t.Errorf("settings %+v", s)
	}
}

func TestStatsSurviveARoundTrip(t *testing.T) {
	s := Settings{Stats: playedStats()}
	data := marshalIndent(t, s)
	back := Parse(data)
	if !reflect.DeepEqual(back, s) {
		t.Errorf("read back %+v from %s", back, data)
	}
	// Spelled out, because a bucket read back empty would still compare equal
	// if the whole tally had been lost.
	if got, want := back.Stats.ForDifficulty(game.Insane).Points, s.Stats.ForDifficulty(game.Insane).Points; got != want {
		t.Errorf("Insane points %d, want %d", got, want)
	}
	if back.Stats.Streak != 3 {
		t.Errorf("streak %d", back.Stats.Streak)
	}
}

func TestTheFileStoresTheStatsByName(t *testing.T) {
	text := marshal(t, Settings{Stats: playedStats()})
	for _, want := range []string{`"stats":{"points":`, `"best_streak":3`, `"by_difficulty":{"Easy":`, `"Insane":{"points":440`} {
		if !strings.Contains(text, want) {
			t.Errorf("%s is missing %s", text, want)
		}
	}
}

func TestAnUnknownDifficultyInTheStatsIsIgnored(t *testing.T) {
	s := Parse([]byte(`{"stats": {"points": 12, "by_difficulty": {"Trivial": {"points": 9}}}}`))
	if s.Stats.Points != 12 {
		t.Errorf("points %d", s.Stats.Points)
	}
	for _, d := range game.Difficulties {
		if p := s.Stats.ForDifficulty(d).Points; p != 0 {
			t.Errorf("%s has %d points", d.Label(), p)
		}
	}
}

func TestGarbageStatsFallBackToAnEmptyTallyWithoutLosingTheRest(t *testing.T) {
	for _, text := range []string{
		`{"theme": "light", "stats": 7}`,
		`{"theme": "light", "stats": "none"}`,
		`{"theme": "light", "stats": {"points": "lots"}}`,
		`{"theme": "light", "stats": []}`,
	} {
		s := Parse([]byte(text))
		if s.Stats != (stats.Stats{}) || s.Theme != Light {
			t.Errorf("%s read as %+v", text, s)
		}
	}
}

// ---------------------------------------------------- the match in flight

func TestAMatchInFlightSurvivesTheTripThroughASnapshot(t *testing.T) {
	saved := savedMatch()
	if back := NewSavedMatch(saved.Snapshot(), saved.MatchPoints); !reflect.DeepEqual(back, saved) {
		t.Errorf("round trip %+v, want %+v", back, saved)
	}
}

func TestAResolvedWordKeepsHowItEnded(t *testing.T) {
	for _, c := range []struct {
		saved string
		want  game.Result
	}{{"won", game.Won}, {"lost", game.Lost}} {
		m := savedMatch()
		m.Result = c.saved
		if got := m.Snapshot().Result; got != c.want {
			t.Errorf("%q read as %v", c.saved, got)
		}
		if back := NewSavedMatch(m.Snapshot(), 0).Result; back != c.saved {
			t.Errorf("%q written back as %q", c.saved, back)
		}
	}
}

func TestTheGuessedLettersAreStoredAsOneReadableString(t *testing.T) {
	if text := marshal(t, savedMatch()); !strings.Contains(text, `"guessed":"AOPT"`) {
		t.Errorf("%s", text)
	}
}

// The theme comes after in_flight on purpose: a key that failed the whole
// parse would take everything, before it and after it alike.
func TestAnUnreadableMatchCostsOnlyItself(t *testing.T) {
	s := Parse([]byte(`{
		"theme": "light",
		"in_flight": { "word": 7 },
		"difficulty": "Insane"
	}`))
	if s.InFlight != nil || s.Theme != Light || game.Difficulty(s.Difficulty) != game.Insane {
		t.Errorf("settings %+v", s)
	}
}

func TestAMatchNeedsNothingButAWord(t *testing.T) {
	m := Parse([]byte(`{ "in_flight": { "word": { "word": "LAPTOP" } } }`)).InFlight
	if m == nil {
		t.Fatal("a word is enough")
	}
	if m.Word.Word != "LAPTOP" || m.Guessed != "" || m.Result != "" || game.Difficulty(m.Difficulty) != game.NoDifficulty {
		t.Errorf("match %+v", m)
	}
}

func TestAMatchWithNoWordInItIsNoMatch(t *testing.T) {
	for _, text := range []string{
		`{ "in_flight": {} }`,
		`{ "in_flight": { "guessed": "AE" } }`,
		`{ "in_flight": [] }`,
		`{ "in_flight": null }`,
	} {
		if m := Parse([]byte(text)).InFlight; m != nil {
			t.Errorf("%s read as %+v", text, m)
		}
	}
}

func TestASettingsFileWrittenBeforeThisExistedSimplyHasNoMatch(t *testing.T) {
	if m := Parse([]byte(`{ "theme": "dark", "difficulty": "Easy" }`)).InFlight; m != nil {
		t.Errorf("match %+v", m)
	}
}

// The hand-drawn gallows is remembered, and a value of the wrong shape costs
// only itself.
func TestTheHandDrawnGallowsIsRemembered(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	Settings{Theme: Light, Sketch: true}.Save(path)
	if back := Load(path); !back.Sketch || back.Theme != Light {
		t.Errorf("loaded %+v", back)
	}
	if s := Parse([]byte(`{"theme": "light", "sketch": "yes"}`)); s.Sketch || s.Theme != Light {
		t.Errorf("parsed %+v", s)
	}
}
