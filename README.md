# hangman-mygo

A Go port of [hangman-gpui](../hangman-gpui), drawn with the native UI of
[mygo](https://github.com/egoist/mygo) instead of gpui-kit. hangman-gpui is
itself a port of [`zerebos/Hangman`](https://github.com/zerebos/Hangman), a
2015 Java/Swing game. The rules, word packs, scoring and alert messages are
the same; the interface was rethought around what mygo does well, which is
making every change move, and the sound was made anew.

## Building and running

You need Go 1.27.3 or later; `go` fetches that toolchain on its own if yours
is older.

```sh
go run .
go test ./...
```

To build the app as people install it (on Windows, `build/windows-amd64/Hangman.exe`
and an installer), use mygo's CLI, which the module pins as a tool:

```sh
go tool mygo build
```

The icon is `resources/icon.png`, built from [`design/icon`](design/icon).

No system libraries are needed on any platform, and the sound works out of
the box, unlike the gpui version, where sound was an opt-in cargo feature.

## The interface

The window is one stage. The header is also the title bar: the wordmark, a
dropdown showing where the words come from (a difficulty, or a list of your
own), the score as chips, a ⋯ menu, and on Windows and Linux the window
controls, drawn at the header's height. Under it, in one centered column, are
the gallows, the guesses left and the letters missed, the word with its
category and clue, and the keyboard.

Almost everything moves:

- A guessed letter flies from its key into every cell it fills, or onto the
  row of missed letters under the gallows.
- The newest body part draws itself on behind a pen tip, and the figure swings
  from the beam as it lands, harder on the guess that loses the word. Giving
  up is a loss like any other: the parts it costs draw on one after another.
- The word shakes on a wrong guess, the newest pip pops, and a played key
  fades into the color its guess earned.
- When a word ends, the keyboard folds away and the result card rises into its
  place. A new word slides in as the old one slides out.
- The lifetime stats open as a drawer that pushes the stage aside, the keys
  shrinking in step with it.
- A match ends on its own summary, its numbers counting up one after another,
  with confetti when you won it.
- Switching between light and dark fades the whole window from one theme to
  the other.

The gallows is drawn by hand: every line wobbles a little, tapers, and runs
past its ends, as if drawn with a marker.

With the desktop's reduce-motion setting on, nothing travels. Fades still
fade in place.

## Sound

Every cue is synthesized by the game itself, in
[`internal/sounds`](internal/sounds), so there are no sound files. The palette
is a soft wooden pluck, like a marimba, on a pentatonic scale:

- A right letter plucks one note higher than the last, so a word being solved
  is heard climbing. The climb starts lower for longer words, so a word's last
  letter always lands on the same note, the one the win resolves to. A guess
  is one note, however many cells it fills.
- A wrong letter is the same wooden bar struck low and damped. Each one falls
  a semitone, and the last guess you can afford lands on the root of the word
  lost phrase, so losing a word ends a fall you have been hearing.
- A hint plays what it is: the low note of the miss it costs, then the note of
  the letter it gives.
- A fresh word opens on a quiet low fifth, and the clue on a quiet high glint.
- A word solved ends on a short rising arpeggio, and a word lost on three soft
  notes stepping down.
- The match's summary opens on a longer phrase if you won it, and a gentle
  close if not.

Giving up sounds like the loss it is, though the original game gave up in
silence. `Sound effects` in the ⋯ menu turns it all off, and is remembered.

## Controls

| Action | How |
| --- | --- |
| Guess a letter | Type it, or click its key |
| Reveal a letter (costs one wrong guess) | `Hint`, or `Ctrl+H` (`⌘H` on macOS) |
| Show what the word means (costs nothing) | `Clue`, or `Ctrl+L` |
| Give up on the word (counts as a loss) | `Give up`, `Ctrl+N`, or the ⋯ menu |
| Next word once a word ends | `New Game`, or Enter / Space |
| See the match once its last word ends | `See the match`, or Enter |
| Play the same difficulty or list again | `Play … again` on the summary, or Enter |
| Change difficulty (starts a fresh match) | The dropdown beside the wordmark |
| Load your own word list or pack | `Ctrl+O`, or the ⋯ menu |
| Show the lifetime stats | The ⋯ menu |
| Throw the lifetime stats away | `Reset stats`, in the stats drawer, which asks first |
| Switch between light and dark | The ⋯ menu |
| Turn the sound effects on or off | The ⋯ menu |

The strip along the bottom of the window lists the shortcuts, dimming the ones
the game would refuse right now. Choosing the difficulty already in play does
nothing mid-match, and switching away from a word you have guessed at counts
it as a loss, as giving up does. The dropdown says so before you choose.

A word list loaded from a file can be played again from its summary for as
long as the game stays open. The list is not saved, so after a relaunch the
summary offers the difficulties instead.

The rules themselves (the guess budget per difficulty, hints, clues, word
packs and scoring) are unchanged from hangman-gpui, whose README explains
each of them in full.

## Settings

Everything but the window is saved to `settings.json` in the platform's
config directory, written the moment it changes:

| Platform | Path |
| --- | --- |
| Windows | `%APPDATA%\hangman-mygo\settings.json` |
| macOS | `~/Library/Application Support/hangman-mygo/settings.json` |
| Linux | `$XDG_CONFIG_HOME/hangman-mygo/settings.json`, or `~/.config/hangman-mygo/settings.json` |

The format is the gpui version's without its `window` key, because mygo
remembers the window's size and place itself. The file is as forgiving as it
was: a key holding nonsense costs only that key, and a saved match that no
real game could have reached is thrown away and a fresh one dealt.

## Layout of the code

| Package | What it is |
| --- | --- |
| [`internal/words`](internal/words) | The word-pack file format: JSON packs with categories and clues, or one word per line. |
| [`internal/game`](internal/game) | The rules: a match of ten words, guesses, hints, giving up, and resuming a saved match. |
| [`internal/stats`](internal/stats) | Points, streaks and the lifetime tally. |
| [`internal/settings`](internal/settings) | `settings.json`: the theme, the difficulty, the stats and the match in flight. |
| [`internal/gallows`](internal/gallows) | The gallows drawing as plain polylines. |
| [`internal/app`](internal/app) | The window: its state and actions (`app.go`), the rules it reads (`rules.go`), and the view, split by part of the window (`view.go`, `header.go`, `stage.go`, `drawer.go`, `summary.go`, `drawing.go`, `confetti.go`, `controls.go`, `motion.go`). |
| [`internal/sounds`](internal/sounds) | The sound cues, synthesized as samples. |
| [`assets`](assets) | The four bundled word packs, embedded in the binary. |

Only `internal/app` and `main.go` import mygo. The other packages are plain
Go, so their tests run without a window. The view's tests run without one
too, through mygo's `ui.Tester`, which clicks, types and advances a clock of
its own, so the animations are tested part way as well as at rest.

To look at the window without opening it, render states and moments part way
through animations to PNGs:

```sh
HANGMAN_SHOTS=./shots go test ./internal/app -run 'TestShots|TestGallowsFrames'
```

## Differences from hangman-gpui

- **One stage instead of a board of panels**, as described above, with the
  scoreboard as header chips, the stats as a drawer, and a summary screen at
  the end of a match.
- **A row of missed letters** under the gallows, which is where a wrong guess
  flies.
- **Play again** for a word list of your own, while the game stays open.
- **A hint never gives the last letter away.** It is refused with one letter
  left, as it is with one guess left, so every word is finished by a letter
  you found. In hangman-gpui a hint could win the word.
- **Giving up spends the rest of the guess budget**, so it is lost like any
  other word, gallows and all.
- **The shortcuts use Command on macOS.** mygo's `ui.Cmd` is Ctrl elsewhere
  and Command on a Mac, where the gpui version used Ctrl everywhere.
- **The window geometry is mygo's.** Its `StateKey` remembers the window, so
  the settings file no longer stores it.
- **New sounds.** The original's two mp3 cues (the loss was a famous arcade
  game's death sound) are replaced by cues the game synthesizes, one for
  every guess as well as for the end of a word and a match, with a switch to
  turn them off.
- **A played key is a tile, not a button.** Clicking it again does nothing,
  where the gpui version's pressed button still took the click.

## Credits

Original game © Zack Rauen ([zerebos](https://github.com/zerebos)), 2015.
Icons after [Lucide](https://lucide.dev) (ISC). The confetti's colors are
canvas-confetti's, by way of mygo's confetti example.
