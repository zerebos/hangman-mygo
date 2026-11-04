package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// drawer is the lifetime stats, in a panel along the right edge beside the
// stage, because the tally is worth reading while you play.
func (a *App) drawer(c *ui.Context, p palette) {
	s := a.session.Stats()
	ui.Column(c.Key("drawer")).Width(drawerW).Shrink(0).Gap(16).Padding(18, 20).
		Background(p.Surface.Alpha(0.5)).BorderWidth(0, 0, 0, 1).BorderColor(p.Border).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Lifetime stats").FontSize(16).Bold()
			ui.Spacer(c)
			if ui.Button(c.Key("close-stats"), "").Label("Close").Tooltip("Close").Children(func() {
				ui.Icon(c, iconClose)
			}).Clicked() {
				a.showStats = false
			}
		})
		tiles := [][3]string{
			{fmt.Sprint(s.Points), "POINTS", "text"},
			{percent(s.WinRate()), "WIN RATE", "text"},
			{fmt.Sprint(s.WordsWon), "WORDS WON", "good"},
			{fmt.Sprint(s.WordsLost), "WORDS LOST", "bad"},
			{fmt.Sprint(s.Streak), "STREAK", "good"},
			{fmt.Sprint(s.BestStreak), "BEST STREAK", "blue"},
		}
		colors := map[string]ui.Color{"text": p.Text, "good": p.Success, "bad": p.Danger, "blue": p.blue}
		for i := 0; i < len(tiles); i += 2 {
			ui.Row(c).Gap(12).Children(func() {
				for _, tile := range tiles[i : i+2] {
					stat(c, p, tile[0], tile[1], colors[tile[2]])
				}
			})
		}
		ui.Row(c).Children(func() {
			stat(c, p, fmt.Sprintf("%d / %d / %d", s.MatchesWon, s.MatchesLost, s.MatchesTied), "MATCHES W / L / T", p.Text)
		})
		ui.Box(c).FillWidth().Height(1).Background(p.Border)
		ui.Column(c).Gap(6).Children(func() {
			breakdownRow(c, func() { eyebrow(c, p, "LEVEL") }, func() {
				for _, h := range []string{"POINTS", "WON", "LOST", "PLAYED"} {
					ui.Box(c).Grow(1).Basis(0).Children(func() { eyebrow(c, p, h) })
				}
			})
			for _, d := range game.Difficulties {
				b := s.ForDifficulty(d)
				breakdownRow(c, func() {
					ui.Text(c, d.Label()).FontSize(13).TextColor(p.TextMuted)
				}, func() {
					for _, cell := range []struct {
						text  string
						color ui.Color
					}{
						{fmt.Sprint(b.Points), p.Text},
						{fmt.Sprint(b.WordsWon), p.Success},
						{fmt.Sprint(b.WordsLost), p.Danger},
						{fmt.Sprint(b.MatchesPlayed()), p.TextMuted},
					} {
						ui.Box(c).Grow(1).Basis(0).Children(func() {
							ui.Text(c, cell.text).FontSize(13).Bold().TextColor(cell.color)
						})
					}
				})
			}
		})
		ui.Spacer(c)
		if ui.Button(c.Key("reset"), "").Label("Reset stats").TextColor(p.Danger).Border(1, p.Danger.Alpha(0.6)).
			Tooltip("Throw away every point, streak and tally").Children(func() {
			ui.Icon(c, iconDelete)
			ui.Text(c, "Reset stats").SingleLine()
		}).Clicked() {
			a.confirmReset = true
		}
	})
}

// stat is one number over its caption.
func stat(c *ui.Context, p palette, value, label string, color ui.Color) {
	ui.Column(c).Grow(1).Basis(0).Gap(2).Padding(10, 12).Radius(10).Background(p.Surface).Children(func() {
		ui.Text(c, value).FontSize(20).Bold().TextColor(color).SingleLine()
		eyebrow(c, p, label)
	})
}

// breakdownRow is one line of the per-difficulty table: a label column wide
// enough for its own heading, then four numbers on a shared grid.
func breakdownRow(c *ui.Context, label, cells func()) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(labelColW).Shrink(0).Children(label)
		cells()
	})
}
