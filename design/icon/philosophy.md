# Icon philosophy

Agreed with the user on 2026-11-04, and what `build.py` executes.

## Seed

Not a gallows: the margin of a school exercise book. Hangman is a pencil-and-paper game, and its
natural habitat is the back page of a notebook during a dull lesson: blue ruled lines, the red
margin rule, a few blanks, and a gallows drawn in whatever pen was to hand. The icon borrows that
page's grammar rather than drawing the subject.

## Position

The icon is a page, not a picture. Its ground is the game's own dark surface, ruled like a
notebook in a faint blue, with the red margin rule down the left. On it, drawn in the same
marker stroke as the game's hand-drawn gallows, sits the frame of the gallows and a rope, and
nothing hanging from it yet: a game about to start, not one lost. Under it, three short blanks,
the first already filled by a single bold letter.

It should feel like something a person drew, which is what the game is. That is the argument for
the one thing the default list would ban: round caps. A marker leaves round ends, and the game
draws with round ends, so the icon does too. Everything else stays strict: two weights, as the game's own
drawing has, two inks plus the ground, and geometry on a grid even where the stroke wobbles.

Hierarchy: the gallows first, the blanks second, the ruled page last and quietest. At 16 px only
the gallows needs to survive; the blanks may merge into a line and the rules may vanish, and that
is fine because they are texture, not message.

The container is the one convention kept on purpose: a rounded square, because macOS requires its
icon shape and Windows and Linux read an icon on a plate as finished. It is the page's edge, not
a badge.

## Hard constraints

```
GRID          8 u base unit; every key coordinate a multiple of it (box 128 u = 1024 px)
BOX           128 × 128 u; plate inset 8 u, corner radius 24 u (macOS-style)
STROKE        two weights, round caps and joins: 6 u for the frame, 4.5 u for the rope and the
              blanks (the game draws its rope lighter than its frame too); rules, margin and
              plate edge are a 1.5 u hairline, the page's own print, not drawn
ANGLES        0°, 90° and the gallows brace at 45°; nothing else
PALETTE       ground #18181b, ink #f4f4f5, rule blue #3b82f6 at 22%, margin red #ef4444 at 70%,
              rope #facc15 (the game's rope, the one accent),
              plate edge #52525b as a 1.5 u hairline, so the plate holds on a dark taskbar
TYPE          none set: the H in the first blank is drawn in three strokes of the light weight
              (a font fell back to its small-size bitmap and filled in at 32 px)
BANNED        gradients, glows, shadows, bevels, 3D; a figure, a noose loop, a skull; letters
              other than the one; any drawn weight but the two above; any colour outside
              the palette; a circle or shield container
```
