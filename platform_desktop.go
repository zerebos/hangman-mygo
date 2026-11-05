//go:build !js

package main

import (
	"runtime"

	"github.com/egoist/mygo"

	"github.com/zerebos/hangman-mygo/internal/app"
	"github.com/zerebos/hangman-mygo/internal/settings"
)

// drawsOwnControls is whether the header draws the window controls at its
// full height. macOS keeps its traffic lights as the system draws them.
const drawsOwnControls = runtime.GOOS != "darwin"

// The settings live in settings.json in the user's config directory.
func loadSettings() settings.Settings  { return settings.Load(settings.Path()) }
func saveSettings(s settings.Settings) { s.Save(settings.Path()) }

// filePicker asks for a word list with the system's open dialog. The dialog
// blocks until the user answers, so it runs on its own goroutine, and the
// file is handed back to the view on the main thread.
func filePicker(window func() *mygo.Window) func(done func(string, error)) {
	return func(done func(string, error)) {
		win := window()
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
}
