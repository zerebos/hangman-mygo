package sounds

import (
	"math"
	"testing"
)

// cues is every cue, by name, as the game asks for them.
func cues() map[string][]float32 {
	return map[string][]float32{
		"right":       Right(0, 1),
		"right high":  Right(14, 1),
		"right x3":    Right(3, 3),
		"right x9":    Right(3, 9),
		"wrong":       Wrong(),
		"hint":        Hint(2, 1),
		"hint x9":     Hint(2, 9),
		"word won":    WordWon(),
		"word lost":   WordLost(),
		"match won":   MatchWon(),
		"match ended": MatchEnded(),
	}
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

// Plucking a letter several times takes longer, a beat per extra cell.
func TestALetterInSeveralCellsPlucksEachOfThem(t *testing.T) {
	one, three := len(Right(0, 1)), len(Right(0, 3))
	if three <= one {
		t.Errorf("three plucks %d samples, one %d", three, one)
	}
}
