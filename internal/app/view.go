package app

import (
	"fmt"
	"runtime"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// The board's measurements, carried over from the gpui version.
const (
	keySize   = 42
	keyGap    = 6
	clueTop   = 12
	stageW    = 332
	columnGap = 20
	cellW     = 34
	cellH     = 38
	wordGap   = 10
	ruleH     = 3
	pipSize   = 9
	labelColW = 88
	// lettersPerRow and indentedRow are the original's letter grid: seven
	// keys a row, with V to Z indented by one key.
	lettersPerRow = 7
	indentedRow   = 3
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
	if a.settings.Theme == settings.Light {
		c.SetTheme(ui.LightTheme())
	} else {
		c.SetTheme(ui.DarkTheme())
	}
	p := paletteFor(c.Theme())

	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	a.handleKeys(c)

	ui.Column(c).Fill().Children(func() {
		a.titleBar(c, p)
		a.toolbar(c, p)
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Padding(columnGap).Gap(columnGap).Children(func() {
			a.playColumn(c, p)
			a.stage(c, p)
		})
		a.shortcutBar(c, p)
	})

	if ui.AlertDialog(c, &a.confirmReset, resetTitle, resetStatsSummary(a.session.Stats()), resetCancel, resetOK) == 1 {
		a.resetStats()
	}
}

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
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		a.giveUp()
	}
	if c.Shortcut(ui.Cmd, ui.KeyO) {
		a.openWordList()
	}
	// A focused button keeps Enter and Space, so these only mean "next word"
	// while the focus is elsewhere.
	if a.game.IsWordOver() && (c.Shortcut(0, ui.KeyEnter) || c.Shortcut(0, ui.KeySpace)) {
		a.nextWord()
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
		return chord("N")
	case shortcutOpenWordList:
		return chord("O")
	}
	return "Enter"
}

// panel is the surface every card in the window is drawn on: the theme's
// surface laid over the background at part alpha, so it sits one step above
// the page in both themes.
func panel(c *ui.Context, p palette) ui.Element {
	return ui.Column(c).Background(p.Surface.Alpha(0.45)).Border(1, p.Border).Radius(10)
}

// eyebrow is a small upper-case section label, the quietest text there is.
func eyebrow(c *ui.Context, p palette, text string) ui.Element {
	return ui.Text(c, text).FontSize(11).Bold().TextColor(p.TextMuted).SingleLine()
}

func (a *App) titleBar(c *ui.Context, p palette) {
	bar := c.TitleBar()
	ui.Row(c).Height(max(bar.Height, 36)).Padding(0, bar.Right+8, 0, bar.Left+16).Gap(8).
		AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Text(c, "Hangman!").FontSize(13).Bold()
		ui.Text(c, subtitle(a.game)).FontSize(12).TextColor(p.TextMuted)
		ui.Spacer(c)
		icon, tip := iconSun, "Switch to light mode"
		if !p.Dark {
			icon, tip = iconMoon, "Switch to dark mode"
		}
		if ui.Button(c.Key("theme"), "").Label("Toggle light and dark mode").Tooltip(tip).Children(func() {
			ui.Icon(c, icon)
		}).Clicked() {
			a.toggleTheme()
		}
	})
}

// iconButton is a toolbar button with an icon before its label.
func iconButton(c *ui.Context, key string, icon *ui.SVG, label string) ui.Element {
	return ui.Button(c.Key(key), "").Label(label).Children(func() {
		ui.Icon(c, icon)
		ui.Text(c, label).SingleLine()
	})
}

func (a *App) toolbar(c *ui.Context, p palette) {
	current := a.game.Difficulty()
	// The pills warn on hover when switching would cost the word in hand.
	// They say "this word" and never the word itself, which is the answer.
	atStake := a.game.HasWordToLose()

	ui.Row(c).Wrap().Gap(12).Padding(10, 20).AlignItems(ui.Center).
		BorderWidth(1, 0, 1, 0).BorderColor(p.Border).Children(func() {
		eyebrow(c, p, "DIFFICULTY")
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
		ui.Spacer(c)

		if iconButton(c, "hint", iconEye, "Hint").Disabled(!a.game.CanHint()).
			Tooltip(hintTooltipFor(a.game) + " (" + chord("H") + ")").Clicked() {
			a.hint()
		}
		if iconButton(c, "clue", iconInfo, "Clue").Disabled(!canShowClue(a.game, a.clueShown)).
			Tooltip(clueTooltipFor(a.game, a.clueShown) + " (" + chord("L") + ")").Clicked() {
			a.showClue()
		}
		ui.Toggle(c.Key("stats"), &a.showStats, "Stats").Tooltip("Show the lifetime score, streak and tally")
		if iconButton(c, "change", iconRotate, "Change Word").
			Tooltip("Give up on this word — counts as a loss (" + chord("N") + ")").Clicked() {
			a.giveUp()
		}
		if iconButton(c, "open", iconFile, "Open word list…").
			Tooltip("Play a word list or pack of your own (" + chord("O") + ")").Clicked() {
			a.openWordList()
		}
	})
}

func (a *App) playColumn(c *ui.Context, p palette) {
	ui.Scroll(c.Key("play")).Grow(1).Gap(12).Children(func() {
		a.scoreboard(c, p)
		a.wordPanel(c, p)
		a.keyboard(c, p)
		if a.game.IsWordOver() {
			a.result(c, p)
		} else {
			a.statusLine(c, p)
		}
		if a.showStats {
			a.statsPanel(c, p)
		}
	})
}

// stat is one number-over-caption tile.
func stat(c *ui.Context, p palette, value, label string, color ui.Color) {
	ui.Column(c).Grow(1).Basis(0).Gap(2).Children(func() {
		ui.Text(c, value).FontSize(20).Bold().TextColor(color).SingleLine()
		eyebrow(c, p, label)
	})
}

func (a *App) scoreboard(c *ui.Context, p palette) {
	s := a.session.Stats()
	panel(c, p).Row().AlignItems(ui.Center).Gap(16).Padding(12, 20).Children(func() {
		divider := func() { ui.Box(c).Width(1).Height(30).Background(p.Border) }
		stat(c, p, fmt.Sprint(a.session.MatchPoints()), "SCORE", p.Text)
		divider()
		stat(c, p, fmt.Sprint(s.Streak), "STREAK", p.Success)
		divider()
		stat(c, p, fmt.Sprint(s.BestStreak), "BEST", p.blue)
		divider()
		stat(c, p, fmt.Sprintf("%d / %d", a.game.WordNumber(), a.game.TotalWords()), "WORD", p.TextMuted)
	})
}

func (a *App) wordPanel(c *ui.Context, p palette) {
	panel(c, p).Gap(8).Padding(12, 20).Children(func() {
		ui.Row(c).AlignItems(ui.Baseline).Gap(12).Children(func() {
			eyebrow(c, p, "THE WORD")
			ui.Spacer(c)
			if category := a.game.Category(); category != "" {
				ui.Text(c, category).FontSize(11).TextColor(p.TextMuted).TextAlign(ui.End).Shrink(1)
			}
		})
		ui.Row(c).Wrap().Justify(ui.Center).Gap(wordGap).Children(func() {
			lost := a.game.Result() == game.Lost
			for i, cell := range a.game.Cells() {
				a.wordCell(c.Key(i), p, cell, lost)
			}
		})
		if clue := a.game.Clue(); a.clueShown && clue != "" {
			ui.Text(c, clue).Margin(clueTop, 0, 0, 0).FontSize(13).Italic().
				TextAlign(ui.Center).TextColor(p.TextMuted)
		}
	})
}

// wordCell is one character of the word: the glyph, and the rule it sits on.
// On a loss the letters the player never guessed are marked, because they
// are the answer rather than their work.
func (a *App) wordCell(c *ui.Context, p palette, cell game.Cell, lost bool) {
	missed := lost && cell.Guessable && !a.game.Guessed().Has(cell.Value)
	glyph, rule := p.Text, p.TextMuted.Alpha(0.4)
	switch {
	case missed:
		glyph, rule = p.Danger, p.Danger.Alpha(0.7)
	case cell.Revealed:
		rule = p.Text.Alpha(0.55)
	}
	width := float32(cellW)
	if !cell.Guessable {
		// A space or a slash is not a blank to fill in.
		width /= 2
	}
	ui.Column(c).Width(width).AlignItems(ui.Center).Gap(6).Children(func() {
		text := " "
		if cell.Revealed {
			text = string(cell.Value)
		}
		ui.Text(c, text).Height(cellH).FontSize(30).Bold().Font("monospace").TextColor(glyph)
		if cell.Guessable {
			ui.Box(c).FillWidth().Height(ruleH).Radius(2).Background(rule)
		}
	})
}

func (a *App) keyboard(c *ui.Context, p palette) {
	panel(c, p).Gap(12).Padding(12, 20).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			eyebrow(c, p, "LETTERS")
			ui.Spacer(c)
			left := a.game.RemainingGuesses()
			s := "es"
			if left == 1 {
				s = ""
			}
			ui.Textf(c, "%d guess%s left", left, s).FontSize(11).TextColor(p.TextMuted)
		})
		// The outer column centers the pad; the inner one sizes to the widest
		// row, so the rows stay left-aligned and the indent reads as one.
		ui.Column(c).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(keyGap).AlignItems(ui.Start).Children(func() {
				for row := 0; row*lettersPerRow < 26; row++ {
					ui.Row(c).Gap(keyGap).Children(func() {
						if row == indentedRow {
							ui.Box(c).Width(keySize)
						}
						for i := row * lettersPerRow; i < min(26, (row+1)*lettersPerRow); i++ {
							a.key(c, p, rune('A'+i))
						}
					})
				}
			})
		})
	})
}

// key is one letter key. A key already played is a plain tile in the color
// its guess earned rather than a button, so clicking it again does nothing.
func (a *App) key(c *ui.Context, p palette, letter rune) {
	label := string(letter)
	switch keyStateOf(a.game, letter) {
	case keyAvailable:
		if ui.Button(c.Key(letter), label).Size(keySize, keySize).Padding(0).Bold().
			Tooltip("Guess " + label).Clicked() {
			a.guess(letter)
		}
	case keyOutOfPlay:
		ui.Button(c.Key(letter), label).Size(keySize, keySize).Padding(0).Bold().Disabled(true)
	default:
		fill := p.Success
		if keyStateOf(a.game, letter) == keyWrong {
			fill = p.Danger
		}
		ui.Box(c.Key(letter)).Size(keySize, keySize).Center().Radius(p.Radius).
			Background(fill.Alpha(0.85)).Border(1, fill).Children(func() {
			ui.Text(c, label).Bold().TextColor(ui.Hex("#ffffff"))
		})
	}
}

func (a *App) statusLine(c *ui.Context, p palette) {
	ui.Row(c).Height(28).Padding(0, 4).AlignItems(ui.Center).Children(func() {
		if n := a.notice; n != nil {
			color := p.Danger
			if n.good {
				color = p.Success
			}
			ui.Text(c, n.text).FontSize(13).Italic().TextColor(color)
		} else {
			ui.Text(c, idleHint).FontSize(13).TextColor(p.TextMuted)
		}
	})
}

// result is the end-of-word panel: what happened, what the word was, and
// what comes next.
func (a *App) result(c *ui.Context, p palette) {
	won := a.game.IsWon()
	headline := gameLost
	if won {
		headline = gameWon
	}
	if a.notice != nil {
		headline = a.notice.text
	}
	tone := p.Danger
	if won {
		tone = p.Success
	}
	panel(c, p).Gap(16).Padding(16).Children(func() {
		ui.Column(c).Gap(4).Padding(12, 16).Radius(8).Background(tone.Alpha(0.12)).Border(1, tone.Alpha(0.5)).
			Children(func() {
				ui.Text(c, headline).FontSize(16).Bold().TextColor(tone)
				ui.Text(c, "The word was "+a.game.Word()).TextColor(p.Text)
			})
		if summary := matchSummary(a.game, a.session); summary != nil {
			// The match is over, so there is no next word to deal. The score
			// takes the button's place.
			color := p.Danger
			if summary.good {
				color = p.Success
			}
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, summary.text).Bold().TextColor(color)
				ui.Text(c, "Pick a difficulty to start a new match.").FontSize(13).TextColor(p.TextMuted)
			})
			return
		}
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, "Press Enter for the next word.").FontSize(13).TextColor(p.TextMuted)
			ui.Spacer(c)
			if ui.PrimaryButton(c.Key("next"), "New Game").Padding(10, 22).Tooltip("Next word (Enter)").Clicked() {
				a.nextWord()
			}
		})
	})
}

// statsPanel is the lifetime tally, folded out under the board. It is inline
// rather than a dialog because it is worth reading while you play.
func (a *App) statsPanel(c *ui.Context, p palette) {
	s := a.session.Stats()
	panel(c, p).Gap(12).Padding(12, 20).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			eyebrow(c, p, "LIFETIME STATS")
			ui.Spacer(c)
			if ui.Button(c.Key("reset"), "").Label("Reset stats").TextColor(p.Danger).Border(1, p.Danger.Alpha(0.6)).
				Tooltip("Throw away every point, streak and tally").Children(func() {
				ui.Icon(c, iconDelete)
				ui.Text(c, "Reset stats").FontSize(12).SingleLine()
			}).Clicked() {
				a.confirmReset = true
			}
		})
		ui.Row(c).Gap(16).Children(func() {
			stat(c, p, fmt.Sprint(s.Points), "POINTS", p.Text)
			stat(c, p, fmt.Sprint(s.WordsWon), "WORDS WON", p.Success)
			stat(c, p, fmt.Sprint(s.WordsLost), "WORDS LOST", p.Danger)
			stat(c, p, percent(s.WinRate()), "WIN RATE", p.Text)
		})
		ui.Row(c).Gap(16).Children(func() {
			stat(c, p, fmt.Sprint(s.Streak), "STREAK", p.Success)
			stat(c, p, fmt.Sprint(s.BestStreak), "BEST STREAK", p.blue)
			stat(c, p, fmt.Sprintf("%d / %d / %d", s.MatchesWon, s.MatchesLost, s.MatchesTied), "MATCHES W / L / T", p.Text)
		})
		ui.Box(c).FillWidth().Height(1).Background(p.Border)
		ui.Column(c).Gap(6).Children(func() {
			breakdownRow(c, func() { eyebrow(c, p, "DIFFICULTY") }, func() {
				for _, h := range []string{"POINTS", "WON", "LOST", "MATCHES"} {
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
}

// breakdownRow is one line of the per-difficulty table: a label column wide
// enough for its own heading, then four numbers on a shared grid.
func breakdownRow(c *ui.Context, label, cells func()) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(labelColW).Shrink(0).Children(label)
		cells()
	})
}

// stage is the gallows and the wrong-guess meter under it.
func (a *App) stage(c *ui.Context, p palette) {
	wrong, budget := a.game.WrongGuesses(), a.game.GuessBudget()
	panel(c, p).Width(stageW).Shrink(0).Gap(12).Padding(16).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			eyebrow(c, p, "THE GALLOWS")
			ui.Spacer(c)
			color := p.Danger
			if wrong == 0 {
				color = p.TextMuted
			}
			ui.Textf(c, "%d / %d", wrong, budget).FontSize(11).Bold().TextColor(color)
		})
		drawGallows(c, p, wrong, budget)
		ui.Row(c).Justify(ui.Center).Gap(6).Children(func() {
			for step := range budget {
				fill := p.TextMuted.Alpha(0.25)
				if step < wrong {
					fill = p.Danger
				}
				ui.Box(c).Size(pipSize, pipSize).Radius(pipSize / 2).Background(fill)
			}
		})
	})
}

// shortcutBar lists the shortcuts along the bottom of the window, outside the
// scrolling column so it never scrolls away.
func (a *App) shortcutBar(c *ui.Context, p palette) {
	ui.Row(c).Wrap().Gap(16).Padding(8, 20).BorderWidth(1, 0, 0, 0).BorderColor(p.Border).Children(func() {
		for _, h := range shortcutLegend(a.game, a.clueShown) {
			entry := ui.Row(c).Gap(6).AlignItems(ui.Center)
			if !h.live {
				entry.Opacity(0.4)
			}
			entry.Children(func() {
				ui.Text(c, h.shortcut.keys()).FontSize(11).Font("monospace").TextColor(p.Text).
					Padding(1, 6).Radius(4).Border(1, p.Border)
				ui.Text(c, h.shortcut.label()).FontSize(11).TextColor(p.TextMuted)
			})
		}
	})
}
