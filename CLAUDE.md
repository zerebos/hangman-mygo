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
- The view runs every frame. State that outlives a frame lives on `App`, never in the view. The
  two exceptions, `drawerShown` and `drawerMoving`, are set by `View` each frame before the stage
  is built, because the keys size themselves from the drawer.

## mygo notes

- One-shot effects are `Keyframes` played once per event (a wrong guess, a guess, a word dealt).
  A sequence replayed through `KeyframeOptions.Run` starts from the value it last showed, so a 0-to-1
  effect is keyed per run instead (`playKey{name, run}`, see `once`) and starts at 0 on its first
  frame. Only sequences that start and end on the same value (the shake, the pip pop) replay with
  `Run`. An element needs a stable key (`c.Key(...)`) to keep its sequences between frames.
- Layout motion is `Transition` on the element. Stage rows use `App.glideOn`, which skips the glide
  while the stats drawer moves; never use `c.DisableAnimations()` for that, because a click builds
  the view twice in a frame and the second pass then stops the drawer's own spring.
- Exits leave as copies that take no room. `AfterExits` is what keeps the keyboard and the result
  card from lying across each other.
- Letters fly with `Transition{Match, Fly}`. A Match pairs one element going with one coming, so a
  key carries hidden stand-ins for each extra cell its letter fills. The match includes the deal,
  so a letter never flies into the next word's keyboard.
- Avoid `Blur` on anything that animates often: it draws into a bitmap every frame, and the
  tester draws on the CPU, which made the view tests six times slower.
- Check motion part way, not only at rest: open the drawer by clicking, `tt.Advance` a little, and
  read the value. A broken animation looks the same once it has finished.
- mygo holds window shortcuts back while a dialog is open, so typing at the Reset question guesses
  nothing. A focused button keeps Enter and Space.
- `mygo.Dialog.Open` blocks, so it runs on a goroutine and returns through `Window.Update`.
- Work that finishes outside the view and wants a toast parks it in `App.toast`, because a toast
  needs a `*ui.Context`.
- On Windows and Linux the header draws the window controls (`HideWindowButtons`); maximize is
  marked `MaximizeButton()` for Windows 11's snap layouts, and the view calls `Maximize` itself.
