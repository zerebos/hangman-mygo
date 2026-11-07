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

- The game also builds for browsers (`GOOS=js GOARCH=wasm`, or `go tool mygo build -platform
  web`). What differs lives in `main`, in `platform_desktop.go` (`!js`) and `platform_web.go`
  (`js`): where the settings go, the file picker, who draws the window controls, whether
  the device is touch-first, and how screen readers hear the game. The app only sees `Options`,
  where a nil `PickFile` means the window cannot open files, so nothing offers to, `Announce`
  reaches a browser's screen readers through a live region, since they cannot see the canvas,
  and `TouchFirst` hides what only a keyboard
  can use (`keysTaught`) until a key is pressed. Check both with `go vet ./... && GOOS=js GOARCH=wasm go vet .`.
- Browsers never give a page Ctrl+N, Ctrl+T or Ctrl+W, so no shortcut may use them.
- A tab can be any size, below the desktop window's minimum, so anything sized from the
  window needs a narrow fallback (`keyLayout`, `wordScale`, `chipsBelowW`, `drawerCoversW`, `boardShapeFor`).
- Only `internal/app` and the `main` package import mygo. `words`, `game`, `stats`, `settings`,
  `gallows` and `sounds` stay plain Go so their tests need no window and no audio device.
- Absent values are zero values: `game.NoDifficulty`, `game.InPlay`, `game.MatchRunning`, an empty
  category or clue, a pack `GuessBudget` of 0. Keep blank meaning absent at every boundary.
- The helpers in `internal/app/rules.go` take no UI types and are tested directly. Anything pulled
  out of the view should keep that shape. `switchDifficulty` and `loadWords` exist because the
  abandon charge depends on the order of statements: read the charge before the reset, and hand it
  back only on success.
- Every action that changes the game or the score ends with `App.persist`, on every guess rather
  than at exit, so killing the process is not a free reroll.
- The view runs every frame. State that outlives a frame lives on `App`, never in the view. The
  exceptions are set by `View` each frame before anything is built: `drawerShown` and
  `drawerMoving`, because the keys size themselves from the drawer, `pointsNow`, because the
  result card and the Score chip both read where a won word's points are, and `bestHeld`,
  because working it out schedules the frame of the Best chip's pop.

## mygo notes

- One-shot effects are `Keyframes` played once per event (a wrong guess, a guess, a word dealt).
  A sequence replayed through `KeyframeOptions.Run` starts from the value it last showed, so a 0-to-1
  effect is keyed per run instead (`playKey{name, run}`, see `once`) and starts at 0 on its first
  frame. Only sequences that start and end on the same value (the shake, the pip pop) replay with
  `Run`. An element needs a stable key (`c.Key(...)`) to keep its sequences between frames.
- Layout motion is `Transition` on the element. Stage rows use `App.glideOn`, which skips the glide
  while the stats drawer moves; never use `c.DisableAnimations()` for that, because a click builds
  the view twice in a frame and the second pass then stops the drawer's own spring.
- The light/dark fade (`themefade.go`) is the one place that does call `c.DisableAnimations()`,
  and it can because the fade is computed from `c.Now()` rather than with an Animate, so nothing
  of its own stops. It needs it because mygo's buttons ease their own colors toward the theme,
  and with the theme changing every frame they would trail the fade. Rule of thumb: disabling
  animations is safe only while the motion that needs it is plain arithmetic on the clock.
- Exits leave as copies that take no room. `AfterExits` is what keeps the keyboard and the result
  card from lying across each other.
- Letters fly with `Transition{Match, Fly}`. A Match pairs one element going with one coming, so a
  key carries hidden stand-ins for each extra cell its letter fills. The match includes the deal,
  so a letter never flies into the next word's keyboard.
- A flight that carries something worth reading flies a copy and leaves the original in place,
  as the result card's points pill does (`pointsPill`), so the motion never takes away what
  the player still wants to read.
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

## Sound

- Every cue is synthesized in `internal/sounds` as mono samples at 48 kHz; there are no sound
  files. The app hands them to an `app.Speaker` (`main.go` wraps mygo's `audio.NewSound`), and
  tests pass a recorder and compare the exact samples, which works because synthesis is
  deterministic.
- Right letters climb a pentatonic scale one new note per distinct letter (`RightStep`), always
  landing on C6 on the last; misses fall one semitone each (`WrongStep`), landing on A2 on the
  budget's last guess. One guess is one note, however many cells it fills. A hint plays the miss
  it costs, then the letter it gives.
- `go run ./.scratch/audition` plays every cue and a scripted word, and writes WAVs to
  `.scratch/audition/out`. It is a local tool: nothing committed may depend on it.
- Every cue must start without a click and have died away before it ends; the tests check the
  first sample and the last 50 ms, not just the last sample, which the closing fade forces quiet.
