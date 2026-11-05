package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// summary is the screen a match ends on: how it went, the match's numbers
// counting up one after another, and the way into the next match.
func (a *App) summary(c *ui.Context, p palette) {
	g := a.game
	headline, tone := "A tie!", p.blue
	switch g.MatchOutcome() {
	case game.MatchWon:
		headline, tone = "You won the match!", p.Success
	case game.MatchLost:
		headline, tone = "You lost the match", p.Danger
	}
	s := a.session.Stats()

	ui.Column(c).Fill().Center().Gap(28).Padding(24).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
			eyebrow(c, p, "MATCH OVER · "+subtitle(g))
			ui.Text(c, headline).FontSize(40).Bold().TextAlign(ui.Center).TextColor(tone)
			if n := matchSummary(g, a.session); n != nil {
				ui.Text(c, n.text).FontSize(14).TextAlign(ui.Center).TextColor(p.TextMuted)
			}
		})
		// Both rows wrap, centered, on a page too narrow for them.
		ui.Row(c).FillWidth().Wrap().Justify(ui.Center).Gap(12).Children(func() {
			tiles := []struct {
				label  string
				value  int
				suffix string
				color  ui.Color
			}{
				{"WORDS WON", g.WordsWon(), fmt.Sprintf(" / %d", g.TotalWords()), p.Success},
				{"WORDS LOST", g.WordsLost(), "", p.Danger},
				{"MATCH POINTS", int(a.session.MatchPoints()), "", p.Text},
				{"STREAK", int(s.Streak), "", p.Success},
				{"BEST STREAK", int(s.BestStreak), "", p.blue},
			}
			for i, tile := range tiles {
				countTile(c.Key(i), p, i, a.deal, tile.label, tile.value, tile.suffix, tile.color)
			}
		})
		ui.Row(c).FillWidth().Wrap().Justify(ui.Center).Gap(10).AlignItems(ui.Center).Children(func() {
			if a.canReplay() {
				if ui.PrimaryButton(c.Key("again"), "Play "+subtitle(g)+" again").Padding(10, 22).
					Tooltip("Start a new match (Enter)").Clicked() {
					a.playAgain()
				}
			}
			for _, d := range game.Difficulties {
				if d == g.Difficulty() {
					continue
				}
				if ui.Button(c.Key(d), d.Label()).Padding(10, 16).Clicked() {
					a.setDifficulty(d)
				}
			}
			if a.canOpenFiles() && ui.Button(c.Key("open"), "Open word list…").Padding(10, 16).Clicked() {
				a.openWordList()
			}
		})
	})
	if g.MatchOutcome() == game.MatchWon {
		confetti(c, a.confetti, a.deal)
	}
}

// countTile is one number of the summary, rising into place and counting up
// from zero, each tile a beat after the one before it.
func countTile(c *ui.Context, p palette, index, run int, label string, value int, suffix string, color ui.Color) {
	tile := ui.Column(c).Width(132).Gap(4).Padding(14, 16).Radius(12).Background(p.Surface)
	t := tile.Keyframes(playKey{"count", run}, []ui.Keyframe{{Value: 0}, {At: 900 * time.Millisecond, Value: 1, Ease: ui.EaseOut}},
		ui.KeyframeOptions{Delay: time.Duration(index) * 140 * time.Millisecond})
	tile.Opacity(min(1, t*4)).Translate(0, 10*(1-min(1, t*2)))
	tile.Children(func() {
		ui.Textf(c, "%d%s", int(float32(value)*t+0.5), suffix).FontSize(26).Bold().TextColor(color).SingleLine()
		eyebrow(c, p, label)
	})
}
