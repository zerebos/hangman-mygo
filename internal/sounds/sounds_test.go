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
			"wrong":       p.Wrong(0),
			"wrong low":   p.Wrong(Top),
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

// One guess is one note, and a word's right letters climb to the octave on
// its last one, its wrong letters fall to the octave below on the last one
// the budget allows, whatever their number.
func TestAWordClimbsToItsOctaveAndMissesFallToTheirs(t *testing.T) {
	for _, n := range []int{1, 2, 3, 6, 10} {
		prev := -1
		for k := range n {
			step := Step(k, n)
			if step < prev {
				t.Errorf("%d guesses: guess %d steps back to %d from %d", n, k, step, prev)
			}
			prev = step
		}
		if prev != Top {
			t.Errorf("%d guesses: the last plays step %d, not %d", n, prev, Top)
		}
	}
	if scaleNote(Top) != root+12 {
		t.Errorf("the top of the climb is note %d, not the octave %d", scaleNote(Top), root+12)
	}
	if missNote(Top) != missRoot-12 {
		t.Errorf("the bottom of the fall is note %d, not the octave %d", missNote(Top), missRoot-12)
	}
	for step := 1; step <= Top; step++ {
		if missNote(step) >= missNote(step-1) {
			t.Errorf("miss %d is note %d, not below %d", step, missNote(step), missNote(step-1))
		}
	}
}
