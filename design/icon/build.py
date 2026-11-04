"""Builds the Hangman app icon on a 128 u grid: a ruled notebook page with an
empty gallows and the first blank filled. See philosophy.md for the position
and its hard constraints.

Needs pycairo, which is easiest in a venv here:

    uv venv .venv && uv pip install --python .venv pycairo
    .venv/Scripts/python build.py
    cp out/icon-1024.png ../../resources/icon.png

mygo makes the platforms' icons from resources/icon.png when it builds.
"""
import math
import os
import random

import cairo

OUT = os.path.join(os.path.dirname(__file__), "out")
U = 8  # px per grid unit at 1024 px
BOX = 128

# Palette, from the game's dark theme.
GROUND = (0x18, 0x18, 0x1B)
INK = (0xF4, 0xF4, 0xF5)
RULE = (0x3B, 0x82, 0xF6, 0.22)
MARGIN = (0xEF, 0x44, 0x44, 0.70)
ROPE = (0xFA, 0xCC, 0x15)
EDGE = (0x52, 0x52, 0x5B)  # the plate's edge, so it holds on a dark taskbar

STROKE = 6      # u, the marker: the frame
LIGHT = 4.5     # u, the lighter line: the rope and the blanks, as the game draws its rope
HAIRLINE = 1.5  # u, the page's print
PLATE = 8       # u inset of the rounded square
RADIUS = 24     # u corner radius

# The drawing, in grid units: every key coordinate on the 8 u grid.
# The beam, the ground and the blanks each sit on a ruled line, as a
# drawing on a real page does: the rules are every 16 u from the beam.
GROUND_Y, POST_X, BEAM_Y, ROPE_X = 72, 48, 24, 96
FRAME = [
    # (points, overshoot at the start, overshoot at the end): only loose
    # ends run past, so the joints meet without knobs.
    ([(32, GROUND_Y), (64, GROUND_Y)], 1.5, 1.5),       # ground
    ([(POST_X, GROUND_Y), (POST_X, BEAM_Y)], 0, 0),     # post
    ([(POST_X, BEAM_Y), (ROPE_X, BEAM_Y)], 0, 1.5),     # beam
    ([(POST_X, 48), (72, BEAM_Y)], 0, 0),               # brace, 45 degrees
]
ROPE_LINE = [(ROPE_X, BEAM_Y), (ROPE_X, 48)]
BLANK_Y = 104
BLANKS = [(32, 48), (56, 72), (80, 96)]  # spanning the frame, 8 u apart
LETTER = "H"
LETTER_BASELINE = 96
LETTER_CAP = 14  # u: the ground and the H each reach half a stroke past their lines, so 16 left only 0.5 x stroke between them
RULES_Y = range(BEAM_Y, BOX, 16)
MARGIN_X = 24


def rgb(c, a=None):
    r, g, b = c[0] / 255, c[1] / 255, c[2] / 255
    alpha = c[3] if len(c) > 3 else 1.0
    return r, g, b, alpha if a is None else a


def rough(points, seed, wobble=0.9, step=4.0, overshoot=1.5, overshoot_end=None):
    """The game's marker line: resampled every few units, pushed off its line
    by two slow waves phased by the seed, and run a little past its ends."""
    (x0, y0), (x1, y1) = points
    length = math.hypot(x1 - x0, y1 - y0)
    dx, dy = (x1 - x0) / length, (y1 - y0) / length
    if overshoot_end is None:
        overshoot_end = overshoot
    x0, y0 = x0 - dx * overshoot, y0 - dy * overshoot
    x1, y1 = x1 + dx * overshoot_end, y1 + dy * overshoot_end
    length += overshoot + overshoot_end
    nx, ny = -dy, dx
    rng = random.Random(seed)
    p1, p2 = rng.uniform(0, 6.28), rng.uniform(0, 6.28)
    n = max(int(length / step), 1)
    out = []
    for k in range(n + 1):
        t = k / n
        s = length * t
        off = wobble * (0.65 * math.sin(s / 23 + p1) + 0.35 * math.sin(s / 7.5 + p2))
        # Ends land true, so the joints of the frame meet.
        off *= min(1.0, t * 6, (1 - t) * 6)
        out.append((x0 + (x1 - x0) * t + nx * off, y0 + (y1 - y0) * t + ny * off))
    return out


def stroke(ctx, pts, width, colour):
    ctx.set_line_cap(cairo.LINE_CAP_ROUND)
    ctx.set_line_join(cairo.LINE_JOIN_ROUND)
    ctx.set_line_width(width)
    ctx.set_source_rgba(*rgb(colour))
    ctx.move_to(*pts[0])
    for p in pts[1:]:
        ctx.line_to(*p)
    ctx.stroke()


def draw(ctx, letter=True, page=True):
    """Draws the icon in grid units (the caller scales the context)."""
    # The plate: the page's edge.
    x, y, w, h, r = PLATE, PLATE, BOX - 2 * PLATE, BOX - 2 * PLATE, RADIUS
    ctx.new_sub_path()
    ctx.arc(x + w - r, y + r, r, -math.pi / 2, 0)
    ctx.arc(x + w - r, y + h - r, r, 0, math.pi / 2)
    ctx.arc(x + r, y + h - r, r, math.pi / 2, math.pi)
    ctx.arc(x + r, y + r, r, math.pi, 3 * math.pi / 2)
    ctx.close_path()
    ctx.set_source_rgba(*rgb(GROUND))
    ctx.fill_preserve()
    ctx.save()
    ctx.clip_preserve()
    # The edge is drawn inside the plate (the clip halves a centred stroke).
    if GROUND[:3] != (0, 0, 0) or len(GROUND) == 3:
        ctx.set_line_width(2 * HAIRLINE)
        ctx.set_source_rgba(*rgb(EDGE))
        ctx.stroke()
    else:
        ctx.new_path()

    if page:
        ctx.set_line_cap(cairo.LINE_CAP_BUTT)
        ctx.set_line_width(HAIRLINE)
        ctx.set_source_rgba(*rgb(RULE))
        for ry in RULES_Y:
            ctx.move_to(0, ry)
            ctx.line_to(BOX, ry)
            ctx.stroke()
        ctx.set_source_rgba(*rgb(MARGIN))
        ctx.move_to(MARGIN_X, 0)
        ctx.line_to(MARGIN_X, BOX)
        ctx.stroke()

    for i, (seg, start, end) in enumerate(FRAME):
        stroke(ctx, rough(seg, 11 + i, overshoot=start, overshoot_end=end), STROKE, INK)
    stroke(ctx, rough(ROPE_LINE, 7, wobble=0.6, overshoot=0, overshoot_end=1.5), LIGHT, ROPE)
    for i, (a, b) in enumerate(BLANKS):
        stroke(ctx, rough([(a, BLANK_Y), (b, BLANK_Y)], 31 + i, wobble=0.5, overshoot=0), LIGHT, INK)

    if letter:
        # The H is drawn, not set: three marker strokes in the light weight,
        # on the grid, its cap height LETTER_CAP over the first blank. A font
        # snapped to its small-size bitmap and filled in solid at 32 px.
        a, b = BLANKS[0]
        cx, top, bottom = (a + b) / 2, LETTER_BASELINE - LETTER_CAP, LETTER_BASELINE
        half = 6
        for i, seg in enumerate([
            [(cx - half, top), (cx - half, bottom)],
            [(cx + half, top), (cx + half, bottom)],
            [(cx - half, (top + bottom) / 2), (cx + half, (top + bottom) / 2)],
        ]):
            stroke(ctx, rough(seg, 51 + i, wobble=0.4, overshoot=0), LIGHT, INK)
    ctx.restore()


def png(path, px, **kw):
    surf = cairo.ImageSurface(cairo.FORMAT_ARGB32, px, px)
    ctx = cairo.Context(surf)
    ctx.scale(px / BOX, px / BOX)
    draw(ctx, **kw)
    surf.write_to_png(path)


def svg(path):
    surf = cairo.SVGSurface(path, BOX * U, BOX * U)
    ctx = cairo.Context(surf)
    ctx.scale(U, U)
    draw(ctx)
    surf.finish()


def mark_only(path, px):
    """The drawn ink alone, on transparent, for the verifier."""
    surf = cairo.ImageSurface(cairo.FORMAT_ARGB32, px, px)
    ctx = cairo.Context(surf)
    ctx.scale(px / BOX, px / BOX)
    global GROUND
    saved = GROUND
    GROUND = (0, 0, 0, 0)
    draw(ctx, page=False)
    GROUND = saved
    surf.write_to_png(path)


if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    png(os.path.join(OUT, "icon-1024.png"), 1024)
    png(os.path.join(OUT, "icon-512.png"), 512)
    svg(os.path.join(OUT, "icon.svg"))
    mark_only(os.path.join(OUT, "mark-1024.png"), 1024)
    print("written to", OUT)
