package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/game"
)

// stage is the scene the game is played on, centered in the window: the
// gallows, the word under it, and the keyboard, which the result card takes
// the place of once the word is over.
func (a *App) stage(c *ui.Context, p palette) {
	// The stage and the match's summary take turns filling the scene: the
	// one going shrinks back and fades as the other comes forward.
	ui.Box(c.Key("scene")).Grow(1).Clip().Children(func() {
		if a.summaryShown {
			ui.Column(c.Key("summary")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Transition(sceneSwap).
				Children(func() { a.summary(c, p) })
			return
		}
		ui.Column(c.Key("play")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Transition(sceneSwap).
			Children(func() { a.board(c, p) })
	})
}

// board is the game itself, in the shape the window has room for (see
// boardShapeFor).
func (a *App) board(c *ui.Context, p palette) {
	w, h := c.Size()
	switch boardShapeFor(w, h) {
	case boardSideBySide:
		ui.Row(c).Fill().Justify(ui.Center).Padding(4, stagePad, 8).Gap(sideGap).Children(func() {
			gallowsW, _ := sideColumns(w)
			ui.Column(c).Width(gallowsW).FillHeight().AlignItems(ui.Center).Gap(10).Children(func() {
				a.scaffold(c, p)
			})
			ui.Column(c).Width(a.stageRoom(c)).FillHeight().Justify(ui.Center).AlignItems(ui.Center).Gap(10).
				Children(func() { a.puzzle(c, p) })
		})
	case boardScrolled:
		// A scroll has no height to give, so the gallows keeps its least.
		ui.Scroll(c.Key("board-scroll")).Fill().AlignItems(ui.Center).Padding(4, stagePad, 12).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(stageW).AlignItems(ui.Center).Gap(10).Children(func() {
				a.scaffold(c, p)
				a.puzzle(c, p)
			})
		})
	default:
		ui.Column(c).Fill().AlignItems(ui.Center).Padding(4, stagePad, 12).Children(func() {
			ui.Column(c).FillHeight().FillWidth().MaxWidth(stageW).AlignItems(ui.Center).Gap(10).Children(func() {
				a.scaffold(c, p)
				a.puzzle(c, p)
			})
		})
	}
}

// scaffold is the gallows and what it has cost so far: the pips and the
// missed letters.
func (a *App) scaffold(c *ui.Context, p palette) {
	a.drawGallows(c, p)
	a.pips(c, p)
	a.missed(c, p)
}

// puzzle is the word and what plays it: the feedback line, and the keyboard
// or the result card.
func (a *App) puzzle(c *ui.Context, p palette) {
	a.word(c, p)
	a.feedback(c, p)
	// The keyboard and the result card take turns in one slot. The one
	// going folds away under the one coming, and the slot springs to the new
	// one's height, which carries the rows above with it.
	a.glideOn(ui.Column(c.Key("bottom")).AlignItems(ui.Center)).Children(func() {
		if a.game.IsWordOver() {
			ui.Column(c.Key("result")).Transition(cardSwap).Children(func() { a.result(c, p) })
		} else {
			ui.Column(c.Key("keys")).AlignItems(ui.Center).Gap(keyGap).Transition(keysSwap).Children(func() {
				a.keyboard(c, p)
				a.actions(c, p)
			})
		}
	})
}

// pips is one dot per wrong guess the budget allows, under the gallows,
// with the count of guesses left beside them.
func (a *App) pips(c *ui.Context, p palette) {
	wrong, budget := a.game.WrongGuesses(), a.game.GuessBudget()
	// It wraps the count under the pips in the narrow gallows column of a
	// short page.
	a.glideOn(ui.Row(c.Key("pips")).FillWidth().Wrap().Justify(ui.Center).AlignItems(ui.Center).Gap(14)).Children(func() {
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
						t := once(halo, "pulse", a.run(wrong), pipPulse, ui.EaseOut)
						grow := 1 + 2*pipHalo/pipSize*t
						halo.Scale(grow, grow)
						halo.Opacity(pipHaloFade * (1 - t))
					}
					dot := ui.Box(c.Key("dot")).Fill().Radius(pipSize / 2).Background(fill)
					// The pip pops as it turns red, so the eye catches which
					// one it was.
					if step+1 == wrong {
						pop := dot.Keyframes("pop", pipPop, ui.KeyframeOptions{Run: uint64(wrong)})
						dot.Scale(pop, pop)
					}
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
		// One line, and wide enough for "10 guesses left", so the row's spring
		// never wraps it as the count loses a digit.
		ui.Textf(c, "%d guess%s left", left, s).FontSize(12).TextColor(color).SingleLine().MinWidth(96)
	})
}

// missed is the letters guessed wrong, as small tiles under the gallows. A
// wrong guess flies here from its key, and the tiles already here make room.
// The row keeps its height while empty, so the first miss moves nothing. It
// wraps onto a second row in the narrow gallows column of a short page.
func (a *App) missed(c *ui.Context, p palette) {
	a.glideOn(ui.Row(c.Key("missed")).FillWidth().MinHeight(26).Gap(6).Wrap().Justify(ui.Center).AlignItems(ui.Center)).Children(func() {
		for r := 'A'; r <= 'Z'; r++ {
			if keyStateOf(a.game, r) != keyWrong {
				continue
			}
			tile := ui.Box(c.Key(r)).Size(26, 26).Center().Radius(6).Label(playedLabel(r, keyWrong)).Background(p.Danger.Alpha(0.15)).
				Border(1, p.Danger.Alpha(0.5))
			if r == a.lastGuess {
				tile.Transition(a.flight(r, 0))
			} else {
				a.glideOn(tile)
			}
			tile.Children(func() {
				ui.Text(c, string(r)).FontSize(13).Bold().TextColor(p.Danger)
			})
		}
	})
}

// word is the puzzle: the category above it, the letters, and the clue
// under it once asked for.
func (a *App) word(c *ui.Context, p palette) {
	// The word has a fresh identity for every word dealt, so a new word
	// slides in as the old one slides out rather than swapping on the spot.
	a.glideOn(ui.Box(c.Key("word-slot"))).Children(func() {
		ui.Column(c.Key(a.deal)).AlignItems(ui.Center).Gap(8).Padding(6, 0).Transition(wordSwap).Children(func() {
			a.wordBody(c, p)
		})
	})
}

func (a *App) wordBody(c *ui.Context, p palette) {
	{
		if category := a.game.Category(); category != "" {
			ui.Text(c, category).FontSize(11).Bold().TextColor(p.TextMuted).Padding(2, 10).
				Radius(999).Border(1, p.Border)
		}
		cells := a.game.Cells()
		scale := wordScale(a.stageRoom(c), cells)
		row := ui.Row(c.Key("word")).Wrap().Justify(ui.Center).Gap(wordGap * scale).Label(wordLabel(a.game.Word()))
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
			for i, cell := range cells {
				fresh := -1
				if a.lastGuess != 0 && cell.Value == a.lastGuess && cell.Revealed {
					fresh, occurrence = occurrence, occurrence+1
				}
				a.wordCell(c.Key(i), p, i, cell, fresh, lost, scale)
			}
		})
		// The clue opens a line for itself, so the rows under it move down
		// to make room instead of jumping.
		if clue := a.game.Clue(); a.clueShown && clue != "" {
			ui.Text(c.Key("clue"), clue).FontSize(14).Italic().TextAlign(ui.Center).TextColor(p.TextMuted).
				MaxWidth(560).Transition(clueReveal)
		}
	}
}

// wordCell is one character of the word: the glyph, and the rule it sits on.
// On a loss the letters the player never guessed are marked, because they
// are the answer rather than their work.
//
// index is the cell's place in the word, which is its place in the win
// reveal. fresh is its place among the cells the last guess turned over, or
// -1 when the last guess did not touch it. scale is how much the word
// shrinks to fit on one line (see wordScale).
func (a *App) wordCell(c *ui.Context, p palette, index int, cell game.Cell, fresh int, lost bool, scale float32) {
	missed := lost && cell.Guessable && !a.game.Guessed().Has(cell.Value)
	glyph, rule := p.Text, p.TextMuted.Alpha(0.4)
	switch {
	case missed:
		glyph, rule = p.Danger, p.Danger.Alpha(0.7)
	case cell.Revealed:
		rule = p.Text.Alpha(0.55)
	}
	width := cellW * scale
	if !cell.Guessable {
		// A space or a slash is not a blank to fill in.
		width /= 2
	}
	ui.Column(c).Width(width).AlignItems(ui.Center).Gap(6).Children(func() {
		text := " "
		if cell.Revealed {
			text = string(cell.Value)
		}
		// A revealed glyph is a new element rather than the hidden one with
		// new text, because a shared element flies only as it appears.
		key := "hidden"
		if cell.Revealed {
			key = "shown"
		}
		g := ui.Text(c.Key(key), text).Height(cellH * scale).FontSize(40 * scale).Bold().Font("monospace").TextColor(glyph)
		// Every cell the last guess filled flies in from its key, the n-th
		// one from the key's n-th stand-in (see key). The rest of a won word
		// resolves left to right as the flourish for finishing it.
		switch {
		case fresh >= 0:
			g.Transition(a.flight(cell.Value, fresh))
		case a.game.IsWon():
			winReveal.play(g, "win", uint64(a.game.WordNumber()), index)
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
	// Full width whatever it says, so its spring never squeezes the text
	// while the line is empty during a result.
	a.glideOn(ui.Row(c.Key("feedback")).FillWidth().Height(24).Justify(ui.Center).AlignItems(ui.Center)).Children(func() {
		text, color, italic := idleHint, p.TextMuted, false
		if !a.keysTaught() {
			text = idleHintTouch
		}
		switch n := a.notice; {
		case a.game.IsWordOver():
			text = ""
		case n != nil:
			text, color, italic = n.text, p.Danger, true
			if n.good {
				color = p.Success
			}
		}
		// One text whose content crossfades as the notice changes.
		line := ui.Text(c.Key("notice"), text).FontSize(13).TextColor(color).SingleLine().
			ContentTransition(ui.ContentTransition{})
		if italic {
			line.Italic()
		}
	})
}

// keyboard is the alphabet in rows, two of thirteen wherever they fit.
func (a *App) keyboard(c *ui.Context, p palette) {
	perRow, _ := a.keyboardLayout(c)
	ui.Column(c).Gap(keyGap).Children(func() {
		for start := 0; start < 26; start += perRow {
			ui.Row(c).Gap(keyGap).Children(func() {
				for i := start; i < min(start+perRow, 26); i++ {
					a.key(c, p, rune('A'+i))
				}
			})
		}
	})
}

// key is one letter key. A key already played is a plain tile in the color
// its guess earned rather than a button, so clicking it again does nothing.
func (a *App) key(c *ui.Context, p palette, letter rune) {
	_, keySize := a.keyboardLayout(c)
	label := string(letter)
	state := keyStateOf(a.game, letter)
	if state == keyAvailable {
		// The key is where its letter flies from once it is guessed: into
		// the word if it is there, and onto the missed row if it is not.
		//
		// A shared element pairs one element going with one coming, so a
		// key can start only one flight. A letter that fills several cells
		// needs one start for each, so the key carries a stand-in for each
		// cell after the first: a copy of its face under it, unseen until
		// the guess takes all of them away at once and each flies to its
		// own cell.
		ui.Box(c.Key(letter)).Size(keySize, keySize).Children(func() {
			for n := 1; n < occurrences(a.game.Word(), letter); n++ {
				ui.Box(c.Key(n)).Absolute().Top(0).Left(0).Size(keySize, keySize).Center().
					Radius(p.Radius).Background(p.Surface).Border(1, p.Border).
					IgnoreInput(true).AccessHidden(true).Transition(a.flight(letter, n)).
					Children(func() { ui.Text(c, label).Bold().FontSize(16) })
			}
			if ui.Button(c.Key("key"), label).Size(keySize, keySize).Padding(0).Bold().FontSize(16).
				Transition(a.flight(letter, 0)).
				Tooltip("Guess " + label).Clicked() {
				a.guess(letter)
			}
		})
		return
	}
	fill := p.Success
	if state == keyWrong {
		fill = p.Danger
	}
	bg, border, text := fill.Alpha(0.85), fill, ui.Hex("#ffffff")
	// A played key is a tile rather than a button, so it says what it is to
	// assistive technology itself: the letter, and what its guess found.
	tile := ui.Box(c.Key(letter)).Size(keySize, keySize).Center().Radius(p.Radius).Label(playedLabel(letter, state))
	// The key the last guess played fades from an unplayed key's colors into
	// the ones its guess earned, rather than snapping to them.
	if letter == a.lastGuess {
		t := once(tile, "settle", a.run(guessCount(a.game)), keySettle, ui.EaseInOut)
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
		if iconButton(c, "give-up", iconFlag, "Give up").
			Tooltip("Give up on this word — counts as a loss (" + chord(giveUpLetter) + ")").Clicked() {
			a.giveUp()
		}
	})
}

// pointsPill is the points the word earned, beside the result's headline.
// They stay for as long as the card does. A copy of their text lies over
// them until it flies to the Score chip (see pointsFlight), and the pill's
// own text stays clear until then, so the two never draw over each other.
func (a *App) pointsPill(c *ui.Context, tone ui.Color) {
	copyShown := a.pointsNow == pointsInCard && !c.Preferences().ReduceMotion
	text := tone
	if copyShown {
		text = tone.Alpha(0)
	}
	label := a.points.label()
	ui.Box(c.Key("points")).Children(func() {
		ui.Text(c.Key("pill"), label).FontSize(20).Bold().TextColor(text).
			Padding(0, 10).Radius(999).Background(tone.Alpha(0.15)).SingleLine()
		if copyShown {
			ui.Text(c.Key("copy"), label).Absolute().Top(0).Left(0).FontSize(20).Bold().
				TextColor(tone).Padding(0, 10).SingleLine().AccessHidden(true).
				Transition(a.pointsInFlight())
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
	ui.Column(c).Width(rowWidth(a.keyboardLayout(c))).Gap(14).Padding(18, 22).Radius(14).
		Background(tone.Alpha(0.1)).Border(1, tone.Alpha(0.45)).Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Text(c, headline).FontSize(20).Bold().TextColor(tone)
				if won && a.points.points > 0 && a.points.deal == a.deal {
					a.pointsPill(c, tone)
				}
			})
			ui.Text(c, "The word was "+a.game.Word()).TextColor(p.Text)
		})
		if a.game.IsMatchOver() {
			// The last word of the match: the button leads to its summary
			// rather than to a next word there is not.
			ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
				ui.Text(c, "That was the last word of the match.").FontSize(13).TextColor(p.TextMuted)
				ui.Spacer(c)
				if ui.PrimaryButton(c.Key("see-match"), "See the match").Padding(10, 22).Tooltip("See the match (Enter)").Clicked() {
					a.showSummary()
				}
			})
			return
		}
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			if a.keysTaught() {
				ui.Text(c, "Press Enter for the next word.").FontSize(13).TextColor(p.TextMuted)
			}
			ui.Spacer(c)
			if ui.PrimaryButton(c.Key("next"), "New Game").Padding(10, 22).Tooltip("Next word (Enter)").Clicked() {
				a.nextWord()
			}
		})
	})
}

// wordLabel names the word's row for screen readers by its length alone,
// which tells them no more than the blanks on screen do.
func wordLabel(word string) string {
	return fmt.Sprintf("The word, %d letters", len([]rune(word)))
}

// playedLabel is what a played key, or a missed letter, says to assistive
// technology: the letter, and whether it was in the word.
func playedLabel(letter rune, state keyState) string {
	if state == keyCorrect {
		return string(letter) + ", in the word"
	}
	return string(letter) + ", not in the word"
}
