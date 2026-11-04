package words

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/zerebos/hangman-mygo/assets"
)

const full = `{
	"name": "Test pack",
	"guess_budget": 8,
	"words": [
		{ "word": "Laptop", "category": "Technology", "clue": "A computer you can close." },
		{ "word": "Add/Drop Form", "category": "College life" },
		{ "word": "Plain" }
	]
}`

func mustParse(t *testing.T, text string) Pack {
	t.Helper()
	pack, err := Parse(text)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return pack
}

func equalWords(t *testing.T, got []Word, want ...Word) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("words %#v, want %#v", got, want)
	}
}

func TestAPackReadsEveryFieldItIsGiven(t *testing.T) {
	pack := mustParse(t, full)
	if pack.Name != "Test pack" || pack.GuessBudget != 8 || len(pack.List) != 3 {
		t.Fatalf("pack %#v", pack)
	}
	if pack.List[0].Category != "Technology" || pack.List[0].Clue != "A computer you can close." {
		t.Errorf("first word %#v", pack.List[0])
	}
}

func TestTheOptionalFieldsAreOptional(t *testing.T) {
	pack := mustParse(t, full)
	if pack.List[1].Clue != "" || pack.List[2].Category != "" || pack.List[2].Clue != "" {
		t.Errorf("words %#v", pack.List)
	}
}

func TestAPackNeedsNothingButItsWords(t *testing.T) {
	pack := mustParse(t, `{ "words": [{ "word": "Alpha" }] }`)
	if pack.Name != "" || pack.GuessBudget != 0 {
		t.Errorf("pack %#v", pack)
	}
	equalWords(t, pack.List, Bare("Alpha"))
}

// The forward half of the no-version-number bargain: a pack written for a
// later game still plays on this one.
func TestAnUnknownKeyIsIgnoredRatherThanRefused(t *testing.T) {
	pack := mustParse(t, `{ "author": "someone", "words": [{ "word": "Alpha", "difficulty": 9 }] }`)
	equalWords(t, pack.List, Bare("Alpha"))
}

// A file that will not parse leaves the word on the board, so it must not look
// like a pack that merely had nothing playable in it.
func TestMalformedJSONIsAnErrorRatherThanAnEmptyPack(t *testing.T) {
	_, err := Parse(`{ "words": [ }`)
	if err == nil || !strings.HasPrefix(err.Error(), "the word pack could not be read") {
		t.Errorf("error %v", err)
	}
}

func TestTextThatDoesNotStartWithABraceIsReadAsLines(t *testing.T) {
	pack := mustParse(t, "Alpha\nOmega\n")
	equalWords(t, pack.List, Bare("Alpha"), Bare("Omega"))
	if pack.Name != "" {
		t.Errorf("name %q", pack.Name)
	}
}

func TestWindowsLineEndingsAreNotPartOfTheWord(t *testing.T) {
	pack := mustParse(t, "Alpha\r\nOmega\r\n")
	equalWords(t, pack.List, Bare("Alpha"), Bare("Omega"))
}

// Leading whitespace must not push a JSON pack down the line branch.
func TestTheFormatIsChosenByTheContentAndNotByAFileName(t *testing.T) {
	pack := mustParse(t, "\n  \t{ \"words\": [{ \"word\": \"Alpha\" }] }")
	equalWords(t, pack.List, Bare("Alpha"))
}

func TestAByteOrderMarkDoesNotTurnAPackIntoAWordList(t *testing.T) {
	pack := mustParse(t, byteOrderMark+"{ \"name\": \"Pets\", \"words\": [{ \"word\": \"Cat\" }] }")
	if pack.Name != "Pets" {
		t.Errorf("name %q", pack.Name)
	}
	equalWords(t, pack.List, Bare("Cat"))
}

func TestABlankCategoryOrClueIsTheSameAsNotHavingOne(t *testing.T) {
	pack := mustParse(t, `{ "words": [{ "word": "Alpha", "category": "", "clue": "   " }] }`)
	equalWords(t, pack.Words(), Bare("ALPHA"))
}

func TestACategoryOrClueWithSomethingInItIsTrimmedRatherThanDropped(t *testing.T) {
	pack := mustParse(t, `{ "words": [{ "word": "Alpha", "category": "  Letters  ", "clue": " The first. " }] }`)
	w := pack.Words()[0]
	if w.Category != "Letters" || w.Clue != "The first." {
		t.Errorf("word %#v", w)
	}
}

func TestAWordKeepsItsSpacesAndPunctuationButLosesItsCase(t *testing.T) {
	equalWords(t, FromWords("  Add/Drop Form  ").Words(), Bare("ADD/DROP FORM"))
}

func TestLinesWithNothingToGuessAreDropped(t *testing.T) {
	pack := mustParse(t, "Alpha\n\n   \n,,,\n123\nOmega\n")
	equalWords(t, pack.Words(), Bare("ALPHA"), Bare("OMEGA"))
}

// Sanitizing rebuilds each word, which is exactly where a field gets quietly
// dropped.
func TestSanitizingKeepsTheCategoryAndClueAttachedToTheWord(t *testing.T) {
	w := mustParse(t, full).Words()[0]
	if w.Word != "LAPTOP" || w.Category != "Technology" || w.Clue != "A computer you can close." {
		t.Errorf("word %#v", w)
	}
}

// Compared with a literal rather than by parsing it back, which would pass
// even if category and clue were swapped both ways at once.
func TestAPackSurvivesARoundTripThroughJSON(t *testing.T) {
	written, err := json.Marshal(mustParse(t, full))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"Test pack","guess_budget":8,"words":[{"word":"Laptop","category":"Technology","clue":"A computer you can close."},{"word":"Add/Drop Form","category":"College life"},{"word":"Plain"}]}`
	if string(written) != want {
		t.Errorf("written %s", written)
	}
}

func TestWhatAWordDoesNotHaveIsLeftOutOfTheJSON(t *testing.T) {
	written, _ := json.Marshal(Bare("ALPHA"))
	if string(written) != `{"word":"ALPHA"}` {
		t.Errorf("written %s", written)
	}
}

func TestABlankNameIsNoName(t *testing.T) {
	if name := (Pack{Name: "   "}).DisplayName(); name != "" {
		t.Errorf("name %q", name)
	}
	if name := (Pack{Name: "  Pets "}).DisplayName(); name != "Pets" {
		t.Errorf("name %q", name)
	}
}

func TestEveryBundledPackParses(t *testing.T) {
	names, err := fs.Glob(assets.Words, "words/*.json")
	if err != nil || len(names) != 4 {
		t.Fatalf("bundled packs %v, %v", names, err)
	}
	for _, name := range names {
		data, _ := assets.Words.ReadFile(name)
		if n := len(MustParseBundled(data).Words()); n != 30 {
			t.Errorf("%s has %d playable words, want 30", name, n)
		}
	}
}
