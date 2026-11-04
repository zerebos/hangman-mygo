# CLAUDE.md

A Go port of `../hangman-gpui` to mygo's native UI (`github.com/egoist/mygo/ui`). Read
`README.md` first. A local checkout of mygo at the pinned commit is at `D:\oss\mygo`, and its
`docs/ui/` is the reference for every widget used here. Verify a mygo API by compiling it or by
grepping `ui/api_gen.go`, where the `Context` and `Element` methods are generated.

## Commands

```sh
go vet ./... && go test ./...
gofmt -l .                      # must print nothing
HANGMAN_SHOTS=<dir> go test ./internal/app -run TestShots   # PNGs of the view
```

## Rules for the code

- Only `internal/app` and `main.go` import mygo. `words`, `game`, `stats`, `settings` and `gallows`
  stay plain Go so their tests need no window.
- Absent values are zero values: `game.NoDifficulty`, `game.InPlay`, `game.MatchRunning`, an empty
  category or clue, a pack `GuessBudget` of 0. Keep blank meaning absent at every boundary.
- The helpers in `internal/app/rules.go` take no UI types and are tested directly. Anything pulled
  out of the view should keep that shape. `switchDifficulty` and `loadWords` exist because the
  abandon charge depends on the order of statements: read the charge before the reset, and hand it
  back only on success.
- Every action that changes the game or the score ends with `App.persist`, on every guess rather
  than at exit, so killing the process is not a free reroll.
- The view runs every frame. State that outlives a frame lives on `App`, never in the view.

## mygo notes

- Animations are one-shot `Keyframes` whose `Run` is a count that moves once per guess (wrong
  guesses, guess count, word number). An element needs a stable key (`c.Key(...)`) to keep its
  sequence between frames.
- mygo holds window shortcuts back while a dialog is open, so no guard is needed for typing at the
  Reset question. A focused button keeps Enter and Space.
- `mygo.Dialog.Open` blocks, so it runs on a goroutine and returns through `Window.Update`.
- Work that finishes outside the view and wants a toast parks it in `App.toast`, because a toast
  needs a `*ui.Context`.
