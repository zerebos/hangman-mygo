//go:build js

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall/js"

	"github.com/egoist/mygo"

	"github.com/zerebos/hangman-mygo/internal/settings"
)

// A tab has no window controls of its own to replace.
const drawsOwnControls = false

// storageKey is the localStorage key the settings are kept under. Go in a
// browser has no files to write, so the page's storage stands in for
// settings.json, holding the same JSON.
const storageKey = "hangman-mygo/settings"

func loadSettings() settings.Settings {
	var data string
	if !withStorage(func(storage js.Value) {
		if v := storage.Call("getItem", storageKey); v.Type() == js.TypeString {
			data = v.String()
		}
	}) || data == "" {
		return settings.Settings{}
	}
	return settings.Parse([]byte(data))
}

func saveSettings(s settings.Settings) {
	data, err := json.Marshal(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hangman: could not save settings: %v\n", err)
		return
	}
	withStorage(func(storage js.Value) { storage.Call("setItem", storageKey, string(data)) })
}

// withStorage runs use on the page's localStorage and reports whether it
// could. A browser may refuse storage outright, as some do in private
// windows, or when it is full, and both throw, which syscall/js turns into
// a panic. The game then plays on without saving.
func withStorage(use func(storage js.Value)) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "hangman: no storage for the settings: %v\n", r)
			ok = false
		}
	}()
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return false
	}
	use(storage)
	return true
}

// Browsers have no open dialog for mygo to show, so there is no Open word
// list on the web.
func filePicker(func() *mygo.Window) func(done func(string, error)) { return nil }

// touchFirst asks the browser whether the device's main pointer is a finger
// that cannot hover, as a phone's or a tablet's is. mygo's input does not
// say which kind of pointer pressed, so this reads the CSS media features
// that say so for the device. A laptop with a touchscreen still has a fine,
// hovering main pointer, so it is not touch-first.
func touchFirst() (yes bool) {
	defer func() {
		if recover() != nil {
			yes = false
		}
	}()
	return js.Global().Call("matchMedia", "(hover: none) and (pointer: coarse)").Get("matches").Bool()
}
