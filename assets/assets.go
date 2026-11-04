// Package assets holds the files baked into the binary: the four bundled word
// packs and the two sound cues.
package assets

import "embed"

// Words holds the bundled packs as words/easy.json, words/med.json,
// words/hard.json and words/insane.json.
//
//go:embed words/*.json
var Words embed.FS

// WinMP3 plays when a word is solved.
//
//go:embed sounds/win.mp3
var WinMP3 []byte

// LossMP3 plays when the gallows is finished.
//
//go:embed sounds/loss.mp3
var LossMP3 []byte
