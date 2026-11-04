package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

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
	ui.Row(c).Height(height).Padding(0, right, 0, bar.Left+20).Gap(16).
		AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Text(c, "Hangman!").FontSize(16).Bold()
		a.difficultyMenu(c)
		// The score sits in the middle of the bar, between who you are and
		// what you can do.
		ui.Spacer(c)
		ui.Row(c).Gap(8).Children(func() {
			s := a.session.Stats()
			chip(c, p, "Score", fmt.Sprint(a.session.MatchPoints()), p.Text)
			chip(c, p, "Streak", fmt.Sprint(s.Streak), p.Success)
			chip(c, p, "Best", fmt.Sprint(s.BestStreak), p.blue)
			chip(c, p, "Word", fmt.Sprintf("%d/%d", a.game.WordNumber(), a.game.TotalWords()), p.TextMuted)
		})
		ui.Spacer(c)
		a.menu(c)
		a.windowControls(c, p, height)
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
		if m.Item("Open word list…").Shortcut(ui.Cmd, ui.KeyO).Chosen() {
			a.openWordList()
		}
		if m.Item("Give up on this word").Shortcut(ui.Cmd, ui.KeyN).Disabled(a.game.IsWordOver()).Chosen() {
			a.giveUp()
		}
		m.Separator()
		if m.Item("Lifetime stats").Checked(a.showStats).Chosen() {
			a.showStats = !a.showStats
		}
		if m.Item("Hand-drawn gallows").Checked(a.settings.Sketch).Chosen() {
			a.settings.Sketch = !a.settings.Sketch
			a.save()
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
