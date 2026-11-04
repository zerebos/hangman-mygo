// Package assets holds the files baked into the binary: the four bundled word
// packs.
package assets

import "embed"

// Words holds the bundled packs as words/easy.json, words/med.json,
// words/hard.json and words/insane.json.
//
//go:embed words/*.json
var Words embed.FS
