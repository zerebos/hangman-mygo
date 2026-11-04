// Hangman is a mygo port of hangman-gpui, itself a port of Zack Rauen's 2015
// Java hangman: the same rules, word packs and layout, drawn by mygo's native
// UI.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/mp3"
	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/assets"
	"github.com/zerebos/hangman-mygo/internal/app"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// cues plays the original's win and loss sounds. A machine with no usable
// audio still plays the game, silently.
type cues struct {
	win, loss *audio.Sound
}

func loadCues() app.Cues {
	win, err := audio.DecodeSound(assets.WinMP3)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hangman: could not decode a sound cue:", err)
		return nil
	}
	loss, err := audio.DecodeSound(assets.LossMP3)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hangman: could not decode a sound cue:", err)
		return nil
	}
	return cues{win, loss}
}

func (c cues) PlayWin()  { c.win.Play() }
func (c cues) PlayLoss() { c.loss.Play() }

func main() {
	path := settings.Path()
	var win *mygo.Window

	// The native picker blocks until the user answers, so it runs on its own
	// goroutine, and the file is handed back to the view on the main thread.
	pickFile := func(done func(string, error)) {
		go func() {
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent: win,
				Title:  "Choose a word list (.txt) or a word pack (.json)",
				Filters: []mygo.FileFilter{
					{Name: "Word lists and packs", Extensions: []string{"txt", "json"}},
					{Name: "All files", Extensions: []string{"*"}},
				},
			})
			if err != nil || len(paths) == 0 {
				return
			}
			contents, err := app.ReadFile(paths[0])
			win.Update(func() { done(contents, err) })
		}()
	}

	game := app.New(settings.Load(path), app.Options{
		SettingsPath: path,
		Cues:         loadCues(),
		PickFile:     pickFile,
	})

	// The header draws its own window controls at its full height, except
	// on macOS, whose traffic lights stay as the system draws them.
	ownControls := runtime.GOOS != "darwin"

	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:             "Hangman!",
			Width:             1000,
			Height:            800,
			MinWidth:          880,
			MinHeight:         660,
			StateKey:          "main",
			TitleBarStyle:     mygo.TitleBarHidden,
			HideWindowButtons: ownControls,
			Content:           ui.View(game.View),
		})
		if ownControls {
			game.SetWindow(win)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
