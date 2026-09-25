#!/usr/bin/env python3
"""Build Howl's icon, mascots and OG image from pixel maps.

The maps below are the design source: one wraith on a 16-cell grid whose
lower rows thin out like the status line's ▒ and ░ dither. Every output is
that grid scaled with no interpolation, so the 16 px favicon and the 512 px
card are the same drawing.

Usage:
    python3 scripts/brand/build-icons.py            # icons and mascots
    python3 scripts/brand/build-icons.py --og DIR   # also site/assets/og.png;
                                                    # DIR holds JetBrainsMono-Regular.ttf and -Bold.ttf

Needs Pillow (the only dependency). Run from the repository root.
"""

import argparse
import math
import pathlib
import sys

from PIL import Image, ImageDraw, ImageFont

ROOT = pathlib.Path(__file__).resolve().parents[2]

BG = (28, 30, 43)  # the site background, #1c1e2b
RGB = {
    "X": (195, 238, 124),  # lime, the context bar's own color
    "E": BG,  # eyes and mouth: the background shows through
    "G": (232, 198, 106),  # gold, the Opus badge color
}
HEX = {"X": "#c3ee7c", "E": "#1c1e2b", "G": "#e8c66a"}

# 14 x 14 cells, so it sits inside a 16-cell canvas with a one-cell margin.
# Rows 11-13 are the dither: the wraith thins out like ▒ and ░.
ICON = """
....XXXXXX....
..XXXXXXXXXX..
.XXXXXXXXXXXX.
.XXXXXXXXXXXX.
XXXEEXXXXEEXXX
XXXEEXXXXEEXXX
XXXEEXXXXEEXXX
XXXXXXXXXXXXXX
XXXXXXXXXXXXXX
XXXXXXXXXXXXXX
XXXXXXXXXXXXXX
X.X.X.X.X.X.X.
.X.X.X.X.X.X.X
X...X...X...X.
""".strip("\n").split("\n")
CANVAS = 16  # cells; the favicon is this grid at one pixel per cell


def grid(width, height):
    return [["."] * width for _ in range(height)]


def paste(canvas, rows, ox, oy):
    for y, r in enumerate(rows):
        for x, c in enumerate(r):
            if c != ".":
                canvas[oy + y][ox + x] = c


def put(canvas, cells, ch):
    for x, y in cells:
        canvas[y][x] = ch


def mascot_listening():
    """The icon wearing headphones, eyes closed: "Howl listens"."""
    c = grid(24, 20)
    paste(c, ICON, 5, 3)
    band = [(x, 0) for x in range(8, 16)] + [(6, 1), (7, 1), (16, 1), (17, 1), (5, 2), (18, 2), (4, 3), (19, 3), (3, 4), (20, 4)]
    put(c, band, "G")
    cups = [(x, y) for y in range(5, 12) for x in (1, 2, 3)] + [(x, y) for y in range(5, 12) for x in (20, 21, 22)]
    put(c, cups, "G")
    # closed eyes: two flat lines where the icon's eyes were
    for y in range(7, 10):
        for x in (8, 9, 14, 15):
            c[y][x] = "X"
    put(c, [(8, 8), (9, 8), (10, 8), (13, 8), (14, 8), (15, 8)], "E")
    return ["".join(r) for r in c]


def mascot_howling():
    """The icon with its mouth open and three gold arcs spreading from it."""
    c = grid(26, 20)
    paste(c, ICON, 2, 2)
    # happy closed eyes: ^ ^
    for y in range(6, 9):
        for x in (5, 6, 11, 12):
            c[y][x] = "X"
    put(c, [(5, 6), (6, 7), (7, 6), (11, 6), (12, 7), (13, 6)], "E")
    put(c, [(5, 7), (13, 7)], "X")
    # mouth, kept above the dither rows so it reads as part of the body
    put(c, [(8, 10), (9, 10), (7, 11), (8, 11), (9, 11), (10, 11), (8, 12), (9, 12)], "E")
    # three arcs fanning out to the right of the body (about 45 degrees each
    # way), one cell thick, leaving a one-cell gap after the body's edge
    cx, cy = 12.5, 11.5
    for r in (5.5, 8.0, 10.5):
        for y in range(20):
            for x in range(17, 26):
                dx, dy = x + 0.5 - cx, y + 0.5 - cy
                if abs(math.hypot(dx, dy) - r) < 0.5 and abs(dy) <= r * 0.7:
                    c[y][x] = "G"
    return ["".join(r) for r in c]


def canvas_for(rows, canvas):
    """Canvas size in cells and the integer offset that centers the sprite."""
    w, h = len(rows[0]), len(rows)
    cw, ch = (canvas, canvas) if canvas else (w + 2, h + 2)
    return cw, ch, (cw - w) // 2, (ch - h) // 2


def render_png(rows, cell, canvas=None, bg=BG, transparent=False):
    """Draw the map at `cell` pixels per cell. `canvas=16` pins the icon grid;
    None gives the sprite a one-cell margin (used for the mascots)."""
    cw, chh, ox, oy = canvas_for(rows, canvas)
    im = Image.new("RGBA", (cw * cell, chh * cell), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    if not transparent:
        # The dark plate: lime alone is 1.3:1 on white, so every image that
        # can land on a light page (favicons, cards, the README mascots) keeps it.
        d.rounded_rectangle([0, 0, cw * cell - 1, chh * cell - 1], radius=round(min(cw, chh) * cell * 0.22), fill=bg)
    draw_cells(d, rows, cell, ox * cell, oy * cell)
    return im


def draw_cells(d, rows, cell, x_off, y_off):
    for y, r in enumerate(rows):
        for x, ch in enumerate(r):
            if ch in RGB:
                x0, y0 = x_off + x * cell, y_off + y * cell
                d.rectangle([x0, y0, x0 + cell - 1, y0 + cell - 1], fill=RGB[ch])


def touch_icon(size=180, cell=11):
    """iOS wants exactly 180 px and masks the icon itself, so this one is an
    opaque square with no rounded corners: 14 cells x 11 px centered."""
    im = Image.new("RGBA", (size, size), BG + (255,))
    d = ImageDraw.Draw(im)
    w, h = len(ICON[0]) * cell, len(ICON) * cell
    draw_cells(d, ICON, cell, (size - w) // 2, (size - h) // 2)
    return im


def render_svg(rows, canvas=CANVAS, bg=True):
    cw, chh, ox, oy = canvas_for(rows, canvas)
    out = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {cw} {chh}" shape-rendering="crispEdges">']
    if bg:
        out.append(f'<rect width="{cw}" height="{chh}" rx="{cw * 0.22:.2f}" fill="#1c1e2b"/>')
    for ch, hx in HEX.items():
        cells = "".join(f'<rect x="{ox + x:g}" y="{oy + y:g}" width="1" height="1"/>' for y, r in enumerate(rows) for x, c in enumerate(r) if c == ch)
        if cells:
            out.append(f'<g fill="{hx}">{cells}</g>')
    out.append("</svg>\n")
    return "".join(out)


def load_fonts(font_dir):
    """Open the OG fonts before anything is written, so a missing TTF stops
    the run with nothing changed instead of leaving a stale og.png behind."""
    try:
        return {
            "bold": ImageFont.truetype(str(font_dir / "JetBrainsMono-Bold.ttf"), 64),
            "wordmark": ImageFont.truetype(str(font_dir / "JetBrainsMono-Bold.ttf"), 48),
            "reg": ImageFont.truetype(str(font_dir / "JetBrainsMono-Regular.ttf"), 24),
            "small": ImageFont.truetype(str(font_dir / "JetBrainsMono-Regular.ttf"), 20),
        }
    except OSError as e:
        sys.exit(f"build-icons: cannot load JetBrains Mono from {font_dir}: {e}")


def build_og(fonts):
    """1200x630 poster: the icon, the tagline, and a four-line statusline."""
    lime, gold, pink, grey, text, muted, sep, blue, panel = (
        "#c3ee7c", "#e8c66a", "#c86791", "#a3a7b8", "#e6e8f0", "#8e93a8", "#5b5d67", "#7ba4e8", "#2f3245")
    bold, wordmark, reg, small = fonts["bold"], fonts["wordmark"], fonts["reg"], fonts["small"]
    im = Image.new("RGB", (1200, 630), BG)
    d = ImageDraw.Draw(im)
    icon = render_png(ICON, 4, CANVAS, transparent=True)  # 64 px, no squircle
    im.paste(icon, (76, 60), icon)
    d.text((150, 68), "Howl", font=wordmark, fill=text)
    d.text((80, 168), "Your AI screams.", font=bold, fill=text)
    d.text((80, 244), "Howl listens.", font=bold, fill=lime)
    d.rounded_rectangle([80, 360, 1120, 560], radius=18, fill=panel)

    def line(y, parts):
        x = 112
        for t, color in parts:
            d.text((x, y), t, font=reg, fill=color)
            x += d.textlength(t, font=reg)

    line(388, [("[Opus 4.6]", gold), (" | ", sep), ("commander@ai-scream.ai", grey), (" | ", sep), ("main*", pink), (" | ", sep), ("$185.4", pink), (" | ", sep), ("91h12m", text)])
    line(428, [("████░░░░░░", lime), ("  41% ( 82K/200K)", text), (" | ", sep), ("█████████░", lime), ("  94% (4h37m/5h)", text)])
    line(468, [("Δ", grey), ("+3.4K", lime), ("/", text), ("-1.3K", pink), (" | ", sep), ("Cache:", grey), ("99%", lime), (" | ", sep), ("Wait:", grey), ("6%", lime), (" | ", sep), ("Cost:", grey), ("$0.03/m", lime), (" | ", sep), ("Insert", lime)])
    line(508, [("Bash", blue), ("(5) ", text), ("Edit", blue), ("(1) ", text), ("Read", blue), ("(3)", text)])
    d.text((80, 586), "Statusline HUD for Claude Code  ·  ai-scream.ai/Howl", font=small, fill=muted)
    im.save(ROOT / "site/assets/og.png", optimize=True)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--og", metavar="FONT_DIR", help="also rebuild site/assets/og.png using the TTFs in FONT_DIR")
    args = ap.parse_args()
    fonts = load_fonts(pathlib.Path(args.og)) if args.og else None

    site = ROOT / "site/assets"
    readme = ROOT / "assets"
    build = ROOT / "build/brand"
    build.mkdir(parents=True, exist_ok=True)

    (site / "favicon.svg").write_text(render_svg(ICON))
    render_png(ICON, 1, CANVAS).save(site / "favicon-16.png")  # one pixel per cell
    render_png(ICON, 2, CANVAS).save(site / "favicon-32.png")
    touch_icon().save(site / "apple-touch-icon.png")  # 180 px, opaque
    render_png(ICON, 4, CANVAS).save(readme / "icon-64.png")
    render_png(ICON, 16, CANVAS).save(readme / "icon-256.png")
    render_png(mascot_listening(), 16).save(readme / "mascot-listening.png")
    render_png(mascot_howling(), 16).save(readme / "mascot-howling.png")
    # 512 px card for the org site; keep the alpha so the corners stay transparent on its light page
    render_png(ICON, 32, CANVAS).save(build / "howl.webp", lossless=True)
    if fonts:
        build_og(fonts)
    print("wrote", site, readme, build)


if __name__ == "__main__":
    sys.exit(main())
