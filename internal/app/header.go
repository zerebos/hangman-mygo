package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// chipsBelowW is the narrowest the bar holds the score chips in its middle.
// A narrower window, which only a browser tab can be, puts them on a row of
// their own under the bar.
const chipsBelowW = 680

// header is the one bar above the stage, and it is the window's title bar
// too: the wordmark, where the words come from, the score as chips, and a
// menu for everything else.
func (a *App) header(c *ui.Context, p palette) {
	bar := c.TitleBar()
	height := max(bar.Height, 52)
	right := bar.Right + 12
	if a.window != nil {
		// Our own controls end the bar, flush with the window's corner.
		right = 0
	}
	w, h := c.Size()
	chipsBelow := w < chipsBelowW
	wordmarkShown := true
	// A short page has no height to spare for a row of chips, so when it is
	// side by side they stay in the bar, and the wordmark makes room.
	if chipsBelow && boardShapeFor(w, h) == boardSideBySide {
		chipsBelow, wordmarkShown = false, false
	}
	ui.Row(c).Height(height).Padding(0, right, 0, bar.Left+20).Gap(16).
		AlignItems(ui.Center).DragWindow().Children(func() {
		if wordmarkShown {
			ui.Text(c, "Hangman!").FontSize(16).Bold()
		}
		a.difficultyMenu(c)
		// The score sits in the middle of the bar, between who you are and
		// what you can do.
		ui.Spacer(c)
		if !chipsBelow {
			a.chips(c, p)
			ui.Spacer(c)
		}
		a.menu(c)
		a.windowControls(c, p, height)
	})
	if chipsBelow {
		ui.Row(c).Justify(ui.Center).Padding(0, 16, 8, 16).Children(func() { a.chips(c, p) })
	}
}

// chips is the scoreboard: the match's score, the streaks and the word.
func (a *App) chips(c *ui.Context, p palette) {
	ui.Row(c).Gap(8).Children(func() {
		s := a.session.Stats()
		a.scoreChip(c, p)
		chip(c, p, "Streak", fmt.Sprint(s.Streak), p.Success)
		popping(c, "best", a.bestPops, func() { chip(c, p, "Best", fmt.Sprint(a.shownBest()), p.blue) })
		chip(c, p, "Word", fmt.Sprintf("%d/%d", a.game.WordNumber(), a.game.TotalWords()), p.TextMuted)
	})
}

// difficultyMenu is a dropdown showing where the words come from: the
// difficulty in play, or the name of a list of your own. Choosing the
// difficulty in play mid-match changes nothing; once the match is over it
// replays the list.
func (a *App) difficultyMenu(c *ui.Context) {
	current := a.game.Difficulty()
	// The warning says "this word" and never the word, which is the answer.
	atStake := a.game.HasWordToLose()
	ui.PopoverMenuButton(c.Key("difficulty"), subtitle(a.game), func(m *ui.Menu) {
		if atStake {
			m.Item("Switching counts this word as a loss").Disabled(true)
			m.Separator()
		}
		for _, d := range game.Difficulties {
			if m.Item(d.Label()).Radio(current == d).Chosen() {
				a.setDifficulty(d)
			}
		}
	}).Tooltip("Difficulty")
}

// scoreChip is the match's score. A copy of the points of a word just won
// flies into it from the result card (see pointsFlight). It shows the score
// without them until the copy lands, then pops, and the number rolls up to
// take them.
func (a *App) scoreChip(c *ui.Context, p palette) {
	score := a.shownScore(a.pointsNow, c.Preferences().ReduceMotion)
	popping(c, "score", a.landings, func() {
		chip(c, p, "Score", fmt.Sprint(score), p.Text)
		// The copy's end of the flight, over the chip's number, while it
		// is on its way. It takes no room, so the chips around it
		// stay put.
		if a.pointsNow == pointsFlying {
			ui.Text(c.Key("points"), a.points.label()).Absolute().Top(4).Right(10).
				FontSize(14).Bold().TextColor(p.Success).SingleLine().AccessHidden(true).Transition(a.pointsInFlight())
		}
	})
}

// popping is a chip that swells once each time runs goes up.
func popping(c *ui.Context, key string, runs int, children func()) {
	box := ui.Box(c.Key(key))
	pop := box.Keyframes("pop", chipPop, ui.KeyframeOptions{Run: uint64(runs)})
	box.Scale(pop, pop).Children(children)
}

// chip is one number of the scoreboard, small enough to live in the header.
func chip(c *ui.Context, p palette, label, value string, color ui.Color) {
	ui.Row(c.Key(label)).AlignItems(ui.Baseline).Gap(6).Padding(5, 12).Radius(999).
		Background(p.Surface).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.TextMuted).SingleLine()
		// The digits roll as they change, as a counter's do.
		ui.Text(c.Key("value"), value).FontSize(14).Bold().TextColor(color).SingleLine().
			ContentTransition(ui.ContentTransition{Numeric: true})
	})
}

// menu holds what the stage has no room for: opening a list, giving up, the
// lifetime stats and the theme. Each item shows the shortcut that does the
// same; handleKeys is what handles the keys.
func (a *App) menu(c *ui.Context) {
	ui.Button(c.Key("menu"), "").Label("More").Tooltip("More").Children(func() {
		ui.Icon(c, iconDots)
	}).PopoverMenuButton(func(m *ui.Menu) {
		if a.canOpenFiles() && m.Item("Open word list…").Shortcut(ui.Cmd, ui.KeyO).Chosen() {
			a.openWordList()
		}
		if m.Item("Give up on this word").Shortcut(ui.Cmd, giveUpKey).Disabled(a.game.IsWordOver()).Chosen() {
			a.giveUp()
		}
		m.Separator()
		if m.Item("Lifetime stats").Checked(a.showStats).Chosen() {
			a.showStats = !a.showStats
		}
		if m.Item("Sound effects").Checked(!a.settings.Mute).Chosen() {
			a.toggleSound()
		}
		theme := "Dark mode"
		if a.settings.Theme == settings.Dark {
			theme = "Light mode"
		}
		if m.Item(theme).Chosen() {
			a.toggleTheme()
		}
	})
}
