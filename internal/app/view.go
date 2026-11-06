package app

import (
	"runtime"

	"github.com/egoist/mygo/ui"
)

// The stage's measurements.
const (
	// maxKey and minKey bound one square letter key, which sizes itself to
	// the room the stage has (see keyLayout).
	maxKey = 46
	minKey = 32
	keyGap = 6
	// cellW and cellH are one character of the word, which is the largest
	// text in the window because it is the puzzle.
	cellW   = 44
	cellH   = 52
	wordGap = 10
	ruleH   = 3
	// minWordScale is the smallest the word shrinks to fit on one line,
	// before it wraps instead.
	minWordScale = 0.6
	pipSize      = 10
	// stageW is the widest the stage grows, so the scene stays one
	// composition on a wide window instead of spreading to its edges.
	stageW  = 720
	drawerW = 320
	// stagePad is the room left and right of the stage.
	stagePad = 24
	// labelColW is the label column of the per-difficulty breakdown.
	labelColW = 60
)

// palette is the theme's colors plus the two the mygo theme has no token
// for: the rope's yellow and the best streak's blue.
type palette struct {
	*ui.Theme
	yellow, blue ui.Color
}

func paletteFor(t *ui.Theme) palette {
	if t.Dark {
		return palette{t, ui.Hex("#facc15"), ui.Hex("#60a5fa")}
	}
	return palette{t, ui.Hex("#ca8a04"), ui.Hex("#2563eb")}
}

// View builds the window from the state, for every frame.
func (a *App) View(c *ui.Context) {
	p := a.theme(c)

	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	a.handleKeys(c)
	a.pointsNow = a.advancePoints(c)
	a.bestHeld = a.advanceBest(c)

	ui.Column(c).Fill().Children(func() {
		a.header(c, p)
		// The stats drawer pushes the stage aside rather than lying over it,
		// so nothing on the board is ever hidden behind it. Its width springs
		// open and shut, and the keys read the same value, so they shrink in
		// step with it rather than jumping when it opens.
		body := ui.Row(c.Key("body")).Grow(1).AlignItems(ui.Stretch)
		target := float32(0)
		if a.showStats {
			target = 1
		}
		a.drawerShown = body.AnimateTo("drawer", target, ui.Smooth)
		a.drawerMoving = !body.Settled("drawer")
		body.Children(func() {
			a.stage(c, p)
			if a.drawerShown > 0.001 {
				a.drawer(c, p)
			}
		})
		a.shortcutBar(c, p)
	})

	if ui.AlertDialog(c, &a.confirmReset, resetTitle, resetStatsSummary(a.session.Stats()), resetCancel, resetOK) == 1 {
		a.resetStats()
	}
}

// stageRoom is how wide the stage's content can be in this frame, beside
// the drawer.
func (a *App) stageRoom(c *ui.Context) float32 {
	w, _ := c.Size()
	w -= drawerW * a.drawerShown
	return min(w, stageW) - 2*stagePad
}

// keyboardLayout is how many letter keys go on a row in this frame, and how
// big each one is.
func (a *App) keyboardLayout(c *ui.Context) (perRow int, size float32) {
	return keyLayout(a.stageRoom(c))
}

// giveUpKey gives up on the word with Cmd (Ctrl). It is G for "give up"
// rather than N, because browsers keep Ctrl+N for a new window and never let
// a page have it.
const (
	giveUpKey    = ui.KeyG
	giveUpLetter = "G"
)

// handleKeys turns the keyboard into guesses and shortcuts. mygo holds the
// window's shortcuts back while a dialog is open, so a letter typed at the
// Reset question is not guessed on the board behind it.
func (a *App) handleKeys(c *ui.Context) {
	if c.Shortcut(ui.Cmd, ui.KeyH) {
		a.hint()
	}
	if c.Shortcut(ui.Cmd, ui.KeyL) {
		a.showClue()
	}
	if c.Shortcut(ui.Cmd, giveUpKey) {
		a.giveUp()
	}
	if c.Shortcut(ui.Cmd, ui.KeyO) {
		a.openWordList()
	}
	// A focused button keeps Enter and Space, so these only mean "next word"
	// while the focus is elsewhere.
	if a.game.IsWordOver() && (c.Shortcut(0, ui.KeyEnter) || c.Shortcut(0, ui.KeySpace)) {
		switch {
		case !a.game.IsMatchOver():
			a.nextWord()
		case a.summaryShown:
			a.playAgain()
		default:
			a.showSummary()
		}
	}
	for i := range 26 {
		key := ui.KeyA + ui.Key(i)
		if c.Shortcut(0, key) || c.Shortcut(ui.Shift, key) {
			a.guess(rune('A' + i))
		}
	}
}

// chord spells a Cmd shortcut the way the platform does.
func chord(letter string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + letter
	}
	return "Ctrl+" + letter
}

func (s shortcut) keys() string {
	switch s {
	case shortcutHint:
		return chord("H")
	case shortcutClue:
		return chord("L")
	case shortcutChangeWord:
		return chord(giveUpLetter)
	case shortcutOpenWordList:
		return chord("O")
	}
	return "Enter"
}

// card is a raised surface: the theme's surface laid over the background at
// part alpha, so it sits one step above the page in both themes.
func card(c *ui.Context, p palette) ui.Element {
	return ui.Column(c).Background(p.Surface.Alpha(0.6)).Border(1, p.Border).Radius(12)
}

// eyebrow is a small upper-case section label, the quietest text there is.
func eyebrow(c *ui.Context, p palette, text string) ui.Element {
	return ui.Text(c, text).FontSize(11).Bold().TextColor(p.TextMuted).SingleLine()
}

// shortcutBar lists the shortcuts along the bottom of the window. It is
// quieter than the stage, but stays, because a tooltip only teaches a
// shortcut to someone who already suspects there is one.
func (a *App) shortcutBar(c *ui.Context, p palette) {
	ui.Row(c).Wrap().Justify(ui.Center).Gap(18).Padding(8, 20).Opacity(0.85).Children(func() {
		legend := shortcutLegend(a.game, legendState{
			clueShown:    a.clueShown,
			summaryShown: a.summaryShown,
			canReplay:    a.canReplay(),
			canOpenFiles: a.canOpenFiles(),
		})
		for _, h := range legend {
			entry := ui.Row(c).Gap(6).AlignItems(ui.Center)
			if !h.live {
				entry.Opacity(0.4)
			}
			entry.Children(func() {
				ui.Text(c, h.shortcut.keys()).FontSize(11).Font("monospace").TextColor(p.TextMuted).
					Padding(1, 6).Radius(4).Border(1, p.Border)
				ui.Text(c, h.shortcut.label()).FontSize(11).TextColor(p.TextMuted)
			})
		}
	})
}
