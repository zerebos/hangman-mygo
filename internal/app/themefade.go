package app

import (
	"reflect"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/zerebos/hangman-mygo/internal/settings"
)

// themeFade is how long the switch between light and dark takes.
const themeFade = 350 * time.Millisecond

// themeState is where the fade between light and dark is: how light the
// window was when the last switch began, when that was, and how light it is
// headed for (0 dark, 1 light).
type themeState struct {
	from, to float32
	at       time.Time
}

// theme sets the frame's theme: light, dark, or a blend on the way from one
// to the other. Every widget and every surface of the stage takes its colors
// from the theme, so fading this one value fades the whole window at once.
//
// The fade is worked out from the clock here rather than with Animate.
// mygo's buttons ease their own colors toward the theme, and with the theme
// changing every frame that easing kept restarting, so the keys held their
// old color until the fade was over and then caught up. While the fade runs,
// the frame disables animations, which sends every widget straight to the
// blended colors; the fade itself is plain arithmetic and goes on. (The
// drawer could not do this, because its own motion is an Animate.)
func (a *App) theme(c *ui.Context) palette {
	target := float32(0)
	if a.settings.Theme == settings.Light {
		target = 1
	}
	now := c.Now()
	switch {
	case a.fade.at.IsZero():
		// The first frame: no fade into the theme the window opens in.
		a.fade = themeState{from: target, to: target, at: now}
	case target != a.fade.to:
		a.fade = themeState{from: a.fade.light(now), to: target, at: now}
	}
	light := a.fade.to
	if !c.Preferences().ReduceMotion {
		light = a.fade.light(now)
	}
	if light != a.fade.to {
		c.AnimationFrame()
		c.DisableAnimations()
	}

	c.SetTheme(blendThemes(ui.DarkTheme(), ui.LightTheme(), light))
	dark, lit := paletteFor(ui.DarkTheme()), paletteFor(ui.LightTheme())
	return palette{
		Theme:  c.Theme(),
		yellow: dark.yellow.MixOklab(lit.yellow, light),
		blue:   dark.blue.MixOklab(lit.blue, light),
	}
}

// light is how light the window is at now.
func (s themeState) light(now time.Time) float32 {
	t := float32(now.Sub(s.at)) / float32(themeFade)
	if t >= 1 {
		return s.to
	}
	t = max(t, 0)
	return s.from + (s.to-s.from)*ui.EaseInOut(t)
}

// blendThemes is the theme t of the way from a to b: every color mixed in
// Oklab, so the midway colors keep their lightness and saturation instead of
// going through a dull gray, and everything else taken from whichever theme
// is nearer. Colors are found by reflection, so a color the theme gains in a
// later mygo blends too.
func blendThemes(a, b *ui.Theme, t float32) *ui.Theme {
	switch {
	case t <= 0:
		return a
	case t >= 1:
		return b
	}
	out := *a
	if t >= 0.5 {
		out = *b
	}
	va, vb, vo := reflect.ValueOf(a).Elem(), reflect.ValueOf(b).Elem(), reflect.ValueOf(&out).Elem()
	colorType := reflect.TypeOf(ui.Color{})
	for i := range vo.NumField() {
		if vo.Field(i).Type() != colorType || !vo.Field(i).CanSet() {
			continue
		}
		ca := va.Field(i).Interface().(ui.Color)
		cb := vb.Field(i).Interface().(ui.Color)
		vo.Field(i).Set(reflect.ValueOf(ca.MixOklab(cb, t)))
	}
	return &out
}
