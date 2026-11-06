package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// drawer is the lifetime stats, in a panel along the right edge beside the
// stage, because the tally is worth reading while you play. On a page too
// narrow to share (see drawerCovers) it covers the stage instead, over a
// shade that closes it when tapped.
func (a *App) drawer(c *ui.Context, p palette) {
	width := float32(drawerW)
	covers := a.drawerCovers(c)
	if covers {
		w, _ := c.Size()
		width = min(width, w)
		// The shade is built before the drawer, because a later sibling
		// draws on top, and the drawer must lie over the shade.
		if ui.Box(c.Key("shade")).Absolute().Top(0).Bottom(0).Left(0).Right(0).
			Background(ui.Hex("#000000").Alpha(0.4 * a.drawerShown)).AccessHidden(true).Clicked() {
			a.showStats = false
		}
	}
	// The outer box is as wide as the drawer is open and clips. The panel in
	// it keeps its full width, so it slides in from the edge rather than
	// being laid out anew at every width on the way.
	box := ui.Box(c.Key("drawer"))
	if covers {
		box.Absolute().Top(0).Bottom(0).Right(0)
	}
	box.Width(width * a.drawerShown).Shrink(0).ClipX().Children(func() {
		a.drawerPanel(c, p, width)
	})
}

func (a *App) drawerPanel(c *ui.Context, p palette, width float32) {
	s := a.session.Stats()
	ui.Column(c).Width(width).FillHeight().Gap(16).Padding(18, 20).
		Background(p.Background.Mix(p.Surface, 0.5)).BorderWidth(0, 0, 0, 1).BorderColor(p.Border).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Lifetime stats").FontSize(16).Bold()
			ui.Spacer(c)
			if ui.Button(c.Key("close-stats"), "").Label("Close").Tooltip("Close").Children(func() {
				ui.Icon(c, iconClose)
			}).Clicked() {
				a.showStats = false
			}
		})
		// The tally scrolls between the title and Reset stats, which stay
		// put, because a phone held sideways is shorter than the tally.
		ui.Scroll(c.Key("tally")).Grow(1).Gap(16).Children(func() {
			tiles := []statTile{
				{"POINTS", counted(s.Points), p.Text},
				{"WIN RATE", func(t float32) string { return percent(s.WinRate() * t) }, p.Text},
				{"WORDS WON", counted(s.WordsWon), p.Success},
				{"WORDS LOST", counted(s.WordsLost), p.Danger},
				{"STREAK", counted(s.Streak), p.Success},
				{"BEST STREAK", counted(s.BestStreak), p.blue},
				{"MATCHES W / L / T", func(t float32) string {
					return counted(s.MatchesWon)(t) + " / " + counted(s.MatchesLost)(t) + " / " + counted(s.MatchesTied)(t)
				}, p.Text},
			}
			// Two tiles a row, and the matches on a row of their own.
			for i := 0; i < len(tiles); i += 2 {
				ui.Row(c).Gap(12).Children(func() {
					for j := i; j < min(i+2, len(tiles)); j++ {
						stat(c, p, j, tiles[j])
					}
				})
			}
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
		})
		if ui.Button(c.Key("reset"), "").Label("Reset stats").TextColor(p.Danger).Border(1, p.Danger.Alpha(0.6)).
			Tooltip("Throw away every point, streak and tally").Children(func() {
			ui.Icon(c, iconDelete)
			ui.Text(c, "Reset stats").SingleLine()
		}).Clicked() {
			a.confirmReset = true
		}
	})
}

// statTile is one number of the drawer: its caption, its value as it reads
// t of the way through counting up (from 0 to 1), and its color.
type statTile struct {
	label string
	value func(t float32) string
	color ui.Color
}

// counted is a number that counts up from zero.
func counted[N uint32 | uint64](n N) func(t float32) string {
	return func(t float32) string { return fmt.Sprint(N(float64(n)*float64(t) + 0.5)) }
}

// Counting up, as the drawer opens: how long one tile takes, and how long
// each waits after the one before it, after statCountDelay for the drawer to
// start sliding in.
const (
	statCount      = 700 * time.Millisecond
	statStagger    = 80 * time.Millisecond
	statCountDelay = 120 * time.Millisecond
)

// stat is one number over its caption. Its number counts up from zero each
// time the drawer opens, a beat after the tile before it, as the summary's
// do: a shut drawer is not built, so each opening builds the tiles afresh
// and their count starts over. Once counted, a number that changes rolls to
// its new value, as the header's chips do.
func stat(c *ui.Context, p palette, index int, tile statTile) {
	box := ui.Column(c.Key(tile.label)).Grow(1).Basis(0).Gap(2).Padding(10, 12).Radius(10).Background(p.Surface)
	t := box.Keyframes("count", []ui.Keyframe{{Value: 0}, {At: statCount, Value: 1, Ease: ui.EaseOut}},
		ui.KeyframeOptions{Delay: statCountDelay + time.Duration(index)*statStagger})
	box.Children(func() {
		value := ui.Text(c.Key("value"), tile.value(t)).FontSize(20).Bold().TextColor(tile.color).SingleLine()
		if t >= 1 {
			value.ContentTransition(ui.ContentTransition{Numeric: true})
		}
		eyebrow(c, p, tile.label)
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
