package sounds

import (
	"math"
	"testing"
)

// cues is every cue of every palette, by name, as the game asks for them.
func cues() map[string][]float32 {
	out := map[string][]float32{}
	for _, p := range []Palette{Wood, Paper} {
		for name, s := range map[string][]float32{
			"right":       p.Right(0, 1),
			"right high":  p.Right(14, 1),
			"right x3":    p.Right(3, 3),
			"right x9":    p.Right(3, 9),
			"wrong":       p.Wrong(),
			"hint":        p.Hint(2, 1),
			"hint x9":     p.Hint(2, 9),
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

// Plucking a letter several times takes longer, a beat per extra cell.
func TestALetterInSeveralCellsPlucksEachOfThem(t *testing.T) {
	for _, p := range []Palette{Wood, Paper} {
		one, three := len(p.Right(0, 1)), len(p.Right(0, 3))
		if three <= one {
			t.Errorf("%s: three plucks %d samples, one %d", p.Name, three, one)
		}
	}
}
