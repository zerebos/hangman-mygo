package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// stage is the scene the game is played on, centered in the window: the
// gallows, the word under it, and the keyboard, which the result card takes
// the place of once the word is over.
func (a *App) stage(c *ui.Context, p palette) {
	ui.Column(c).Grow(1).AlignItems(ui.Center).Padding(4, stagePad, 12).Children(func() {
		ui.Column(c).FillHeight().FillWidth().MaxWidth(stageW).AlignItems(ui.Center).Gap(10).Children(func() {
			drawGallows(c, p, a.game.WrongGuesses(), a.game.GuessBudget())
			a.pips(c, p)
			a.word(c, p)
			a.feedback(c, p)
			if a.game.IsWordOver() {
				a.result(c, p)
			} else {
				a.keyboard(c, p)
				a.actions(c, p)
			}
		})
	})
}

// pips is one dot per wrong guess the budget allows, under the gallows,
// with the count of guesses left beside them.
func (a *App) pips(c *ui.Context, p palette) {
	wrong, budget := a.game.WrongGuesses(), a.game.GuessBudget()
	ui.Row(c).AlignItems(ui.Center).Gap(14).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			for step := range budget {
				fill := p.TextMuted.Alpha(0.25)
				if step < wrong {
					fill = p.Danger
				}
				ui.Box(c.Key(step)).Size(pipSize, pipSize).Children(func() {
					// The pip the last wrong guess filled pulses: a halo
					// behind it grows past it and fades out, so it ends
					// invisible, and being absolute it moves nothing else.
					if step+1 == wrong {
						halo := ui.Box(c.Key("halo")).Absolute().Top(0).Left(0).Size(pipSize, pipSize).
							Radius(pipSize / 2).Background(p.Danger)
						t := once(halo, "pulse", wrong, pipPulse, ui.EaseOut)
						grow := 1 + 2*pipHalo/pipSize*t
						halo.Scale(grow, grow)
						halo.Opacity(pipHaloFade * (1 - t))
					}
					ui.Box(c).Fill().Radius(pipSize / 2).Background(fill)
				})
			}
		})
		left := a.game.RemainingGuesses()
		s := "es"
		if left == 1 {
			s = ""
		}
		color := p.TextMuted
		if left <= 2 && !a.game.IsWordOver() {
			color = p.Danger
		}
		ui.Textf(c, "%d guess%s left", left, s).FontSize(12).TextColor(color)
	})
}

// word is the puzzle: the category above it, the letters, and the clue
// under it once asked for.
func (a *App) word(c *ui.Context, p palette) {
	ui.Column(c).AlignItems(ui.Center).Gap(8).Padding(6, 0).Children(func() {
		if category := a.game.Category(); category != "" {
			ui.Text(c, category).FontSize(11).Bold().TextColor(p.TextMuted).Padding(2, 10).
				Radius(999).Border(1, p.Border)
		}
		row := ui.Row(c.Key("word")).Wrap().Justify(ui.Center).Gap(wordGap)
		// A wrong guess shakes the word. Below one wrong guess there is no
		// shake at all, so the row does not twitch as a word is dealt.
		if wrong := a.game.WrongGuesses(); wrong > 0 {
			row.Translate(row.Keyframes("shake", shakeFrames, ui.KeyframeOptions{Run: uint64(wrong)}), 0)
		}
		row.Children(func() {
			lost := a.game.Result() == game.Lost
			// A letter's occurrences stagger against each other rather than
			// against the word, so guessing the only E of a long word does
			// not wait out the cells in front of it.
			occurrence := 0
			for i, cell := range a.game.Cells() {
				fresh := -1
				if a.lastGuess != 0 && cell.Value == a.lastGuess && cell.Revealed {
					fresh, occurrence = occurrence, occurrence+1
				}
				a.wordCell(c.Key(i), p, i, cell, fresh, lost)
			}
		})
		if clue := a.game.Clue(); a.clueShown && clue != "" {
			ui.Text(c, clue).FontSize(14).Italic().TextAlign(ui.Center).TextColor(p.TextMuted).MaxWidth(560)
		}
	})
}

// wordCell is one character of the word: the glyph, and the rule it sits on.
// On a loss the letters the player never guessed are marked, because they
// are the answer rather than their work.
//
// index is the cell's place in the word, which is its place in the win
// reveal. fresh is its place among the cells the last guess turned over, or
// -1 when the last guess did not touch it.
func (a *App) wordCell(c *ui.Context, p palette, index int, cell game.Cell, fresh int, lost bool) {
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
		g := ui.Text(c.Key("glyph"), text).Height(cellH).FontSize(40).Bold().Font("monospace").TextColor(glyph)
		// The win comes first: the guess that wins also reveals letters, and
		// taking the win means the row plays one flourish, not two at once.
		switch {
		case a.game.IsWon():
			winReveal.play(g, "win", uint64(a.game.WordNumber()), index)
		case fresh >= 0:
			guessReveal.play(g, "guess", uint64(guessCount(a.game)), fresh)
		}
		if cell.Guessable {
			ui.Box(c).FillWidth().Height(ruleH).Radius(2).Background(rule)
		}
	})
}

// feedback is the line under the word: the latest notice, or the idle hint
// while there is nothing to report. A finished word says its piece in the
// result card instead.
func (a *App) feedback(c *ui.Context, p palette) {
	ui.Row(c).Height(24).AlignItems(ui.Center).Children(func() {
		if a.game.IsWordOver() {
			return
		}
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

// keyboard is the alphabet on two rows of thirteen.
func (a *App) keyboard(c *ui.Context, p palette) {
	ui.Column(c).Gap(keyGap).Children(func() {
		for row := range 2 {
			ui.Row(c).Gap(keyGap).Children(func() {
				for i := row * keysPerRow; i < (row+1)*keysPerRow; i++ {
					a.key(c, p, rune('A'+i))
				}
			})
		}
	})
}

// key is one letter key. A key already played is a plain tile in the color
// its guess earned rather than a button, so clicking it again does nothing.
func (a *App) key(c *ui.Context, p palette, letter rune) {
	keySize := a.keySize(c)
	label := string(letter)
	state := keyStateOf(a.game, letter)
	if state == keyAvailable {
		if ui.Button(c.Key(letter), label).Size(keySize, keySize).Padding(0).Bold().FontSize(16).
			Tooltip("Guess " + label).Clicked() {
			a.guess(letter)
		}
		return
	}
	fill := p.Success
	if state == keyWrong {
		fill = p.Danger
	}
	bg, border, text := fill.Alpha(0.85), fill, ui.Hex("#ffffff")
	tile := ui.Box(c.Key(letter)).Size(keySize, keySize).Center().Radius(p.Radius)
	// The key the last guess played fades from an unplayed key's colors into
	// the ones its guess earned, rather than snapping to them.
	if letter == a.lastGuess {
		t := once(tile, "settle", guessCount(a.game), keySettle, ui.EaseInOut)
		bg, border, text = p.Surface.MixOklab(bg, t), p.Border.MixOklab(border, t), p.Text.MixOklab(text, t)
	}
	tile.Background(bg).Border(1, border).Children(func() {
		ui.Text(c, label).Bold().FontSize(16).TextColor(text)
	})
}

// actions is the row under the keyboard: the two ways to get help, and
// giving up.
func (a *App) actions(c *ui.Context, p palette) {
	ui.Row(c).Gap(8).Padding(6, 0, 0).Children(func() {
		if iconButton(c, "hint", iconEye, "Hint").Disabled(!a.game.CanHint()).
			Tooltip(hintTooltipFor(a.game) + " (" + chord("H") + ")").Clicked() {
			a.hint()
		}
		if iconButton(c, "clue", iconInfo, "Clue").Disabled(!canShowClue(a.game, a.clueShown)).
			Tooltip(clueTooltipFor(a.game, a.clueShown) + " (" + chord("L") + ")").Clicked() {
			a.showClue()
		}
		if iconButton(c, "change", iconRotate, "Change word").
			Tooltip("Give up on this word — counts as a loss (" + chord("N") + ")").Clicked() {
			a.giveUp()
		}
	})
}

// iconButton is a button with an icon before its label.
func iconButton(c *ui.Context, key string, icon *ui.SVG, label string) ui.Element {
	return ui.Button(c.Key(key), "").Label(label).Children(func() {
		ui.Icon(c, icon)
		ui.Text(c, label).SingleLine()
	})
}

// result is the card that takes the keyboard's place once the word is over:
// what happened, what the word was, and what comes next.
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
	keySize := a.keySize(c)
	ui.Column(c).Width(keysPerRow*keySize+(keysPerRow-1)*keyGap).Gap(14).Padding(18, 22).Radius(14).
		Background(tone.Alpha(0.1)).Border(1, tone.Alpha(0.45)).Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, headline).FontSize(20).Bold().TextColor(tone)
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
