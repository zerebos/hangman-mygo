// Package words is the word-pack file format and nothing else.
//
// A pack is a named list of words, each of which may carry a category and a
// clue. The four bundled packs are JSON files embedded by package assets, and
// the player can load one of their own from disk:
//
//	{
//	  "name": "Easy",
//	  "words": [
//	    { "word": "Laptop", "category": "Technology", "clue": "A computer you can close." },
//	    { "word": "Add/Drop Form", "category": "College life" }
//	  ]
//	}
//
// Everything but "word" is optional. Keys this version does not know are
// ignored rather than refused, which is encoding/json's default, so a pack
// written for a later version still plays here. That is why the format has no
// version number.
//
// The original's plain-text lists are still legal: Parse reads anything that
// does not start with `{` as one word per line.
package words

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// Word is one word and everything the pack knows about it.
//
// An empty Category or Clue means the pack said nothing about it. Sanitize
// turns a blank one into an empty one, so nothing downstream has to tell
// "said nothing" from "said nothing useful".
type Word struct {
	// Word is the text to guess. Spaces, slashes and apostrophes are fine;
	// only ASCII letters are guessable and everything else shows through.
	Word string `json:"word"`
	// Category is what kind of thing it is, as "Food". It shows while the word
	// is in play, so it must never give the answer away.
	Category string `json:"category,omitempty"`
	// Clue is a sentence about the word's meaning, shown only when asked for.
	Clue string `json:"clue,omitempty"`
}

// Bare is a word with nothing known about it but its text, which is what a
// line of a .txt becomes.
func Bare(word string) Word {
	return Word{Word: word}
}

// normalized is the word as the game plays it: trimmed and uppercased, with a
// blank category or clue made empty.
func (w Word) normalized() Word {
	return Word{
		Word:     asciiUpper(strings.TrimSpace(w.Word)),
		Category: strings.TrimSpace(w.Category),
		Clue:     strings.TrimSpace(w.Clue),
	}
}

// playable reports whether there is anything in the word to guess. A line of
// punctuation or digits would otherwise become a word with nothing in it.
func (w Word) playable() bool {
	for _, r := range w.Word {
		if IsLetter(r) {
			return true
		}
	}
	return false
}

// Pack is a named list of words, as read from a file. Nothing in it is
// sanitized until Words is called.
type Pack struct {
	// Name is what to call the list on screen. It is empty for a .txt and for
	// a JSON pack that leaves it out; the view has its own wording then.
	Name string `json:"name,omitempty"`
	// GuessBudget is how many wrong guesses the pack wants to allow, or 0 for
	// no opinion. Only a pack the player loads is asked, and the game clamps
	// it into what the gallows can draw.
	GuessBudget int `json:"guess_budget,omitempty"`
	// List is the words in file order. A match draws from them at random.
	List []Word `json:"words"`
}

// Parse reads a pack from the contents of a file the player picked.
//
// The format is chosen by what the text is rather than by the file's name:
// text whose first non-blank character is `{` is parsed as JSON, and anything
// else is read one word per line. Only the JSON branch can fail.
//
// A leading byte-order mark comes off first. Windows editors write one by
// default, and with it left on a JSON pack fails the `{` test and silently
// loads its own lines as words.
func Parse(text string) (Pack, error) {
	text = strings.TrimPrefix(text, "Feff")
	if strings.HasPrefix(strings.TrimLeftFunc(text, unicode.IsSpace), "{") {
		var pack Pack
		if err := json.Unmarshal([]byte(text), &pack); err != nil {
			return Pack{}, &PackError{err}
		}
		return pack, nil
	}
	return FromLines(text), nil
}

// FromLines is a pack read from plain text, one word per line, with no
// categories or clues.
func FromLines(text string) Pack {
	lines := strings.Split(text, "\n")
	// A final newline ends the last line rather than starting an empty one.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	list := make([]Word, len(lines))
	for i, line := range lines {
		list[i] = Bare(strings.TrimSuffix(line, "\r"))
	}
	return Pack{List: list}
}

// FromWords is a pack of words already in hand.
func FromWords(words ...string) Pack {
	list := make([]Word, len(words))
	for i, word := range words {
		list[i] = Bare(word)
	}
	return Pack{List: list}
}

// MustParseBundled reads one of the packs baked into the binary, and panics
// if it does not parse.
//
// That is the opposite of how package settings treats its file, on purpose: a
// settings file we cannot read is the player's, and falling back costs them
// nothing, but a pack we cannot read is ours, and shipping one is a bug. A
// test parses all four.
func MustParseBundled(data []byte) Pack {
	var pack Pack
	if err := json.Unmarshal(data, &pack); err != nil {
		panic(fmt.Sprintf("a bundled word pack must parse: %v", err))
	}
	return pack
}

// DisplayName is the pack's name with its surrounding blanks gone, so a pack
// named "   " has no name at all.
func (p Pack) DisplayName() string {
	return strings.TrimSpace(p.Name)
}

// Words is the pack's words, trimmed, uppercased, and with the unplayable
// ones dropped.
func (p Pack) Words() []Word {
	return Sanitize(p.List)
}

// Sanitize trims and uppercases words and drops the ones with nothing to
// guess. A saved match read back from settings.json goes through this too,
// because the player may have edited that file by hand.
func Sanitize(list []Word) []Word {
	out := make([]Word, 0, len(list))
	for _, w := range list {
		if n := w.normalized(); n.playable() {
			out = append(out, n)
		}
	}
	return out
}

// IsLetter reports whether r is one of the guessable letters, A to Z in
// either case.
func IsLetter(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

// asciiUpper uppercases the ASCII letters of s and leaves every other rune as
// it is, so an "é" in a word stays an "é" that shows through.
func asciiUpper(s string) string {
	return strings.Map(func(r rune) rune {
		if 'a' <= r && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, s)
}

// PackError is a JSON pack that would not parse. It wraps the decoder's own
// error, which says what was wrong and where.
type PackError struct {
	err error
}

func (e *PackError) Error() string {
	return "the word pack could not be read: " + e.err.Error()
}

func (e *PackError) Unwrap() error {
	return e.err
}
