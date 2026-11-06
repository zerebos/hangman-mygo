package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/settings"
)

// openStats opens the stats drawer on a window of w by h, once anything the
// resize set moving has settled, and says where the A key was before and
// after.
func openStats(t *testing.T, w, h int) (a *App, tt *ui.Tester, before, after float32) {
	t.Helper()
	a, tt, _ = tester(t, settings.Settings{})
	tt.SetSize(w, h)
	tt.Advance(2 * time.Second)
	keyX := func() float32 {
		r, ok := tt.Find("A")
		if !ok {
			t.Fatal("no A key")
		}
		return r.X
	}
	before = keyX()
	a.showStats = true
	tt.Frame()
	tt.Advance(2 * time.Second)
	return a, tt, before, keyX()
}

// On a page too narrow to share, the drawer lies over the stage, so the
// keys stay where they were, and the shade beside it closes it.
func TestTheDrawerCoversTheStageOnANarrowPage(t *testing.T) {
	for _, w := range []int{390, drawerCoversW - 1} {
		a, tt, before, after := openStats(t, w, 844)
		if before != after {
			t.Errorf("%d px: the A key moved from %v to %v under the drawer", w, before, after)
		}
		// The shade shows on the strip of stage left of the drawer.
		tt.ClickAt(10, 300)
		tt.Advance(2 * time.Second)
		if a.showStats {
			t.Errorf("%d px: tapping the shade left the drawer open", w)
		}
	}
}

// A window wide enough to share keeps the drawer beside the stage, which
// moves aside for it.
func TestTheDrawerSharesAWideWindow(t *testing.T) {
	for _, w := range []int{drawerCoversW, 1000} {
		a, tt, before, after := openStats(t, w, 800)
		if before == after {
			t.Errorf("%d px: the stage did not move aside for the drawer", w)
		}
		tt.ClickAt(10, 300)
		tt.Frame()
		if !a.showStats {
			t.Errorf("%d px: a click on the stage closed the drawer", w)
		}
	}
}

// The shade darkens the stage, not the drawer over it, so the drawer's own
// background is the same color covering a phone as beside a wide window.
func TestTheShadeLiesUnderTheDrawer(t *testing.T) {
	panel := func(w int) [4]uint8 {
		_, tt, _, _ := openStats(t, w, 844)
		// A point in the panel's padding, by the right edge.
		px := tt.Image().RGBAAt(w-8, 400)
		return [4]uint8{px.R, px.G, px.B, px.A}
	}
	if narrow, wide := panel(390), panel(1000); narrow != wide {
		t.Errorf("the drawer is %v over a phone's stage and %v beside a wide one", narrow, wide)
	}
}
