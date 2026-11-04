package sounds

import (
	"math"
	"testing"
)

// cues is every cue of every palette, by name, as the game asks for them.
func cues() map[string][]float32 {
	out := map[string][]float32{}
	for _, p := range []Palette{Wood} {
		for name, s := range map[string][]float32{
			"right":       p.Right(0),
			"right top":   p.Right(Top),
			"right low":   p.Right(RightStep(0, 14)),
			"wrong":       p.Wrong(WrongStep(0, 10)),
			"wrong low":   p.Wrong(0),
			"hint":        p.Hint(2),
			"hint top":    p.Hint(Top),
			"word won":    p.WordWon(),
			"word lost":   p.WordLost(),
			"match won":   p.MatchWon(),
			"match ended": p.MatchEnded(),
		} {
			out[p.Name+" "+name] = s
		}
	}
	return out
}

// Every cue is short, never clips, starts without a click and ends in
// silence, so cues can overlap and cut each other off without popping.
func TestEveryCueIsShortCleanAndQuietAtItsEnds(t *testing.T) {
	for name, s := range cues() {
		if len(s) == 0 || len(s) > 4*Rate {
			t.Errorf("%s: %d samples", name, len(s))
			continue
		}
		peak := float32(0)
		for _, v := range s {
			if math.IsNaN(float64(v)) {
				t.Fatalf("%s: NaN", name)
			}
			peak = max(peak, float32(math.Abs(float64(v))))
		}
		if peak > 0.8 || peak < 0.2 {
			t.Errorf("%s: peak %v", name, peak)
		}
		if first := math.Abs(float64(s[0])); first > 0.01 {
			t.Errorf("%s: starts at %v, which clicks", name, first)
		}
		// The last sample is forced quiet by the fade; what matters is that
		// the cue had died away before it, or the fade chops a ringing note.
		tail := s[len(s)-Rate/20:]
		loud := float32(0)
		for _, v := range tail {
			loud = max(loud, float32(math.Abs(float64(v))))
		}
		if loud > 0.05*peak {
			t.Errorf("%s: still at %.0f%% of its peak in its last 50 ms", name, 100*loud/peak)
		}
	}
}

// The right-letter pluck climbs the scale step by step, through octaves.
func TestTheRightLetterClimbs(t *testing.T) {
	prev := 0
	for step := range 12 {
		note := scaleNote(step)
		if note <= prev {
			t.Errorf("step %d is note %d, not above %d", step, note, prev)
		}
		prev = note
	}
	if scaleNote(5) != root+12 {
		t.Errorf("step 5 is %d, want the octave %d", scaleNote(5), root+12)
	}
}

// Every right letter of a word climbs one new note and the last lands on
// the octave, and every miss falls one semitone and the last affordable one
// lands on the bottom, however many there are.
func TestRightLettersClimbAndMissesFall(t *testing.T) {
	for _, n := range []int{1, 2, 3, 6, 10, 14} {
		for k := 1; k < n; k++ {
			if scaleNote(RightStep(k, n)) <= scaleNote(RightStep(k-1, n)) {
				t.Errorf("%d letters: letter %d does not climb", n, k)
			}
			if WrongStep(k, n) != WrongStep(k-1, n)-1 {
				t.Errorf("%d misses: miss %d does not fall a semitone", n, k)
			}
		}
		if got := scaleNote(RightStep(n-1, n)); got != root+12 {
			t.Errorf("%d letters: the last plays note %d, not the octave %d", n, got, root+12)
		}
		if got := WrongStep(n-1, n); got != 0 {
			t.Errorf("%d misses: the last is %d semitones above the bottom", n, got)
		}
	}
	// Insane's six guesses start the fall on D3.
	if missBottom+WrongStep(0, 6) != 50 {
		t.Errorf("Insane's first miss is note %d, not D3", missBottom+WrongStep(0, 6))
	}
}
