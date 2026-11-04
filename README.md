# hangman-mygo

A Go port of [hangman-gpui](../hangman-gpui), drawn with the native UI of
[mygo](https://github.com/egoist/mygo) instead of gpui-kit. hangman-gpui is
itself a port of [`zerebos/Hangman`](https://github.com/zerebos/Hangman), a
2015 Java/Swing game, so this keeps the same rules, word packs, layout, sound
cues and alert messages.

This is phase one, a faithful port. Phase two reworks the interface around
what mygo does well.

## Building and running

You need Go 1.27.3 or later; `go` fetches that toolchain on its own if yours
is older.

```sh
go run .
go test ./...
```

No system libraries are needed on any platform, and the sound cues work out
of the box, unlike the gpui version, where sound was an opt-in cargo feature.

## Layout

| Package | What it is |
| --- | --- |
| [`internal/words`](internal/words) | The word-pack file format: JSON packs with categories and clues, or one word per line. |
| [`internal/game`](internal/game) | The rules: a match of ten words, guesses, hints, giving up, and resuming a saved match. |
| [`internal/stats`](internal/stats) | Points, streaks and the lifetime tally. |
| [`internal/settings`](internal/settings) | `settings.json`: the theme, the difficulty, the stats and the match in flight. |
| [`internal/gallows`](internal/gallows) | The gallows drawing as plain polylines. |
| [`internal/app`](internal/app) | The window: its state, its actions and the mygo view. |
| [`assets`](assets) | The four bundled packs and the two mp3 cues, embedded in the binary. |

Only `internal/app` and `main.go` import mygo. The other packages are plain
Go, so their tests run without a window. The view's tests run without one
too, through mygo's `ui.Tester`, which clicks and types the way a player
would.

To look at the window without opening it, render a few states to PNGs:

```sh
HANGMAN_SHOTS=./shots go test ./internal/app -run TestShots
```

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

## Controls

| Action | How |
| --- | --- |
| Guess a letter | Type it, or click its key |
| Reveal a letter (costs one wrong guess) | `Hint`, or `Ctrl+H` (`⌘H` on macOS) |
| Show what the word means (costs nothing) | `Clue`, or `Ctrl+L` |
| Next word once a word ends | `New Game`, or Enter / Space |
| Give up on the word (counts as a loss) | `Change Word`, or `Ctrl+N` |
| Load your own word list or pack | `Open word list…`, or `Ctrl+O` |
| Change difficulty (starts a fresh match) | The Easy / Medium / Hard / Insane pills |
| Show the lifetime stats | `Stats` |
| Throw the lifetime stats away | `Reset stats`, inside the stats panel, which asks first |

The rules themselves, the guess budget per difficulty, hints, clues, word
packs and scoring, are unchanged from hangman-gpui, whose README explains
each of them in full.

## Differences from hangman-gpui

- **The shortcuts use Command on macOS.** mygo's `ui.Cmd` is Ctrl elsewhere
  and Command on a Mac, where the gpui version used Ctrl everywhere.
- **The window geometry is mygo's.** Its `StateKey` remembers the window, so
  the settings file no longer stores it.
- **Sound is on by default**, through mygo's own audio package.
- **A played key is a tile, not a button.** Clicking it again does nothing,
  where the gpui version's pressed button still took the click.
- **Toolbar tooltips spell the shortcut out in text**, as `(Ctrl+H)`, rather
  than as a key chip read from the keymap.

## Credits

Original game © Zack Rauen ([zerebos](https://github.com/zerebos)), 2015.
Toolbar icons after [Lucide](https://lucide.dev) (ISC).
