package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// header is the one bar above the stage, and it is the window's title bar
// too: the wordmark and where the words come from, the difficulty, the
// score as chips, and a menu for everything else.
func (a *App) header(c *ui.Context, p palette) {
	bar := c.TitleBar()
	ui.Row(c).Height(max(bar.Height, 52)).Padding(0, bar.Right+12, 0, bar.Left+20).Gap(16).
		AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Row(c).AlignItems(ui.Baseline).Gap(8).Children(func() {
			ui.Text(c, "Hangman!").FontSize(16).Bold()
			ui.Text(c, subtitle(a.game)).FontSize(12).TextColor(p.TextMuted).SingleLine()
		})
		a.difficultyPills(c)
		ui.Spacer(c)
		s := a.session.Stats()
		chip(c, p, "Score", fmt.Sprint(a.session.MatchPoints()), p.Text)
		chip(c, p, "Streak", fmt.Sprint(s.Streak), p.Success)
		chip(c, p, "Best", fmt.Sprint(s.BestStreak), p.blue)
		chip(c, p, "Word", fmt.Sprintf("%d/%d", a.game.WordNumber(), a.game.TotalWords()), p.TextMuted)
		a.menu(c)
	})
}

// difficultyPills is the four difficulties as joined toggles. A click on
// the one in play mid-match changes nothing; once the match is over it
// replays the list.
func (a *App) difficultyPills(c *ui.Context) {
	current := a.game.Difficulty()
	// The warning says "this word" and never the word, which is the answer.
	atStake := a.game.HasWordToLose()
	ui.ToggleGroup(c, func() {
		for _, d := range game.Difficulties {
			on := current == d
			pill := ui.Toggle(c.Key(d), &on, d.Label())
			if atStake && current != d {
				pill.Tooltip("Switching now counts this word as a loss")
			}
			if pill.Changed() {
				a.setDifficulty(d)
			}
		}
	}).Label("Difficulty")
}

// chip is one number of the scoreboard, small enough to live in the header.
func chip(c *ui.Context, p palette, label, value string, color ui.Color) {
	ui.Row(c.Key(label)).AlignItems(ui.Baseline).Gap(6).Padding(5, 12).Radius(999).
		Background(p.Surface).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.TextMuted).SingleLine()
		ui.Text(c, value).FontSize(14).Bold().TextColor(color).SingleLine()
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
		theme := "Dark mode"
		if a.settings.Theme == settings.Dark {
			theme = "Light mode"
		}
		if m.Item(theme).Chosen() {
			a.toggleTheme()
		}
	})
}
