// Hangman is a mygo port of hangman-gpui, itself a port of Zack Rauen's 2015
// Java hangman: the same rules, word packs and layout, drawn by mygo's native
// UI.
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/app"
	"github.com/zerebos/hangman-mygo/internal/sounds"
)

// speaker plays the game's cues through mygo's audio, which mixes them on
// its own goroutine. A machine with no usable audio still plays the game,
// silently.
type speaker struct{}

func (speaker) Play(samples []float32, delay time.Duration) {
	sound := audio.NewSound(sounds.Rate, 1, samples)
	if delay <= 0 {
		sound.Play()
		return
	}
	time.AfterFunc(delay, func() { sound.Play() })
}

// Where the settings live, how to ask for a file, and who draws the window
// controls differ between desktops and browsers: see platform_desktop.go and
// platform_web.go.
func main() {
	var win *mygo.Window
	game := app.New(loadSettings(), app.Options{
		Save:       saveSettings,
		Speaker:    speaker{},
		PickFile:   filePicker(func() *mygo.Window { return win }),
		TouchFirst: touchFirst(),
		Announce:   announcer(),
	})

	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:             "Hangman!",
			Width:             1000,
			Height:            800,
			MinWidth:          880,
			MinHeight:         660,
			StateKey:          "main",
			TitleBarStyle:     mygo.TitleBarHidden,
			HideWindowButtons: drawsOwnControls,
			Content:           ui.View(game.View),
		})
		if drawsOwnControls {
			game.SetWindow(win)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
