#!/usr/bin/env python3
"""Render the README screenshots from the real binary.

Feeds two fixed sessions (normal mode and danger mode) into the built
`howl`, then draws its ANSI output with JetBrains Mono, so the pictures in
the README show exactly what the current renderer prints and can be
regenerated whenever it changes.

The binary also reads ~/.claude.json (account) and ~/.claude/hud/config.json
(preset), runs git in the workspace, and tails the transcript. All of that
comes from a temporary HOME and a temporary repository built here, so no
real account or path ends up in a public image.

Usage:
    python3 scripts/brand/render-statusline.py --howl build/howl --fonts FONT_DIR [--out assets]

FONT_DIR holds JetBrainsMono-Regular.ttf and JetBrainsMono-Bold.ttf. The
plain-text version of each screenshot is printed so it can be pasted into
the README's accessibility blocks.
"""

import argparse
import json
import os
import pathlib
import re
import subprocess
import sys
import tempfile
import time

from PIL import Image, ImageDraw, ImageFont

# The terminal theme the README screenshots use, lightened where the site
# did the same for contrast.
PANEL = (47, 50, 69)
FG = (216, 219, 230)
COLORS = {
    "31": (200, 103, 145),  # red → pink
    "32": (195, 238, 124),  # green → lime
    "33": (232, 198, 106),  # yellow → gold
    "34": (123, 164, 232),  # blue
    "35": (200, 103, 145),  # magenta → pink
    "36": (108, 203, 217),  # cyan
    "38;5;208": (255, 135, 0),  # orange
    "38;5;245": (163, 167, 184),  # grey
}
DIM = (151, 155, 173)

SGR = re.compile(r"\x1b\[([0-9;]*)m")
OSC = re.compile(r"\x1b\]8;;[^\x1b]*\x1b\\")

EMAIL = "commander@ai-scream.ai"
TRANSCRIPT = [
    {"message": {"content": [{"id": f"t{i}", "name": name, "type": "tool_use"}]}}
    for i, name in enumerate(["Bash"] * 5 + ["Edit"] + ["Read"] * 3)
] + [
    {"message": {"content": [{"id": "task_1", "input": {"description": "Explore the codebase", "subagent_type": "explore"}, "name": "Task", "type": "tool_use"}]}},
]


def session(now, *, percent, total_cost, minutes, api_minutes, output_tokens, cache_read, input_tokens):
    """One stdin payload: a 200K window; the quotas reset 4h37m and 2d21h from now."""
    used = 200_000 * percent // 100
    return {
        "session_id": "readme",
        "version": "2.1.272",
        "model": {"id": "claude-opus-4-6", "display_name": "Opus 4.6"},
        "cost": {
            "total_cost_usd": total_cost,
            "total_duration_ms": minutes * 60_000,
            "total_api_duration_ms": api_minutes * 60_000,
            "total_lines_added": 3412,
            "total_lines_removed": 1287,
        },
        "context_window": {
            "total_input_tokens": used,
            "total_output_tokens": 4521,
            "context_window_size": 200_000,
            "used_percentage": percent,
            "remaining_percentage": 100 - percent,
            "current_usage": {
                "input_tokens": input_tokens,
                "output_tokens": output_tokens,
                "cache_creation_input_tokens": 0,
                "cache_read_input_tokens": cache_read,
            },
        },
        "rate_limits": {
            "five_hour": {"used_percentage": 6, "resets_at": now + 4 * 3600 + 37 * 60},
            "seven_day": {"used_percentage": 21, "resets_at": now + 2 * 86400 + 21 * 3600},
        },
        "vim": {"mode": "INSERT"},
    }


def make_home(tmp):
    home = tmp / "home"
    (home / ".claude" / "hud").mkdir(parents=True)
    (home / ".claude.json").write_text(json.dumps({"oauthAccount": {"emailAddress": EMAIL}}))
    (home / ".claude" / "hud" / "config.json").write_text(json.dumps({"preset": "full"}))
    return home


def make_repo(tmp):
    repo = tmp / "Howl"
    repo.mkdir()
    env = {"HOME": str(tmp), "PATH": os.environ["PATH"]}
    env.update({k: v for k, v in (("GIT_AUTHOR_NAME", "howl"), ("GIT_AUTHOR_EMAIL", EMAIL), ("GIT_COMMITTER_NAME", "howl"), ("GIT_COMMITTER_EMAIL", EMAIL))})
    run = lambda *a: subprocess.run(["git", *a], cwd=repo, env=env, check=True, capture_output=True)
    run("init", "-q", "-b", "main")
    (repo / "main.go").write_text("package main\n")
    run("add", "main.go")
    run("commit", "-q", "-m", "init")
    (repo / "main.go").write_text("package main\n\nfunc main() {}\n")  # dirty, so the branch shows as main*
    return repo


def run_howl(howl, home, payload):
    env = {"HOME": str(home), "PATH": os.environ["PATH"], "COLUMNS": "120", "TERM": "xterm-256color"}
    out = subprocess.run([str(howl)], input=json.dumps(payload), capture_output=True, text=True, env=env, check=True)
    return [line.replace(" ", " ") for line in out.stdout.rstrip("\n").split("\n")]


def spans(line):
    """Split one ANSI line into (text, color, bold) runs."""
    line = OSC.sub("", line)
    color, bold, dim = FG, False, False
    pos, runs = 0, []
    for m in SGR.finditer(line):
        if m.start() > pos:
            runs.append((line[pos:m.start()], DIM if dim else color, bold))
        params = m.group(1) or "0"
        codes = [params] if params.startswith("38;5;") else params.split(";")
        for code in codes:
            if code in ("", "0"):
                color, bold, dim = FG, False, False
            elif code == "1":
                bold = True
            elif code == "2":
                dim = True
            elif code in COLORS:
                color = COLORS[code]
        pos = m.end()
    if pos < len(line):
        runs.append((line[pos:], DIM if dim else color, bold))
    return runs


def plain(line):
    return SGR.sub("", OSC.sub("", line))


def render(lines, fonts, scale=2):
    size, pad, lh = 14 * scale, 16 * scale, int(14 * scale * 1.75)
    regular = ImageFont.truetype(str(fonts / "JetBrainsMono-Regular.ttf"), size)
    bold = ImageFont.truetype(str(fonts / "JetBrainsMono-Bold.ttf"), size)
    cell = regular.getlength("M")
    width = int(max(sum(regular.getlength(t) for t, _, _ in spans(l)) for l in lines)) + 2 * pad
    im = Image.new("RGBA", (width, len(lines) * lh + 2 * pad), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    d.rounded_rectangle([0, 0, width - 1, im.height - 1], radius=8 * scale, fill=PANEL)
    for i, line in enumerate(lines):
        x, y = pad, pad + i * lh + (lh - size) // 2
        for text, color, is_bold in spans(line):
            font = bold if is_bold else regular
            for ch in text:
                if ch == "\U0001f534":  # 🔴 has no glyph in JetBrains Mono; draw the dot
                    r = size * 0.42
                    d.ellipse([x + cell - r, y + size / 2 - r + 2, x + cell + r, y + size / 2 + r + 2], fill=(226, 75, 74))
                    x += cell * 2
                    continue
                d.text((x, y), ch, font=font, fill=color)
                x += font.getlength(ch)
    return im


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--howl", required=True, type=pathlib.Path)
    ap.add_argument("--fonts", required=True, type=pathlib.Path)
    ap.add_argument("--out", default="assets", type=pathlib.Path)
    args = ap.parse_args()
    for f in ("JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf"):
        if not (args.fonts / f).exists():
            sys.exit(f"render-statusline: {f} not found in {args.fonts}")
    if not args.howl.exists():
        sys.exit(f"render-statusline: {args.howl} not found; run make build")

    now = int(time.time())
    with tempfile.TemporaryDirectory() as t:
        tmp = pathlib.Path(t)
        home, repo = make_home(tmp), make_repo(tmp)
        transcript = tmp / "transcript.jsonl"
        transcript.write_text("".join(json.dumps(e) + "\n" for e in TRANSCRIPT))
        common = {"transcript_path": str(transcript), "cwd": str(repo), "workspace": {"current_dir": str(repo), "project_dir": str(repo)}}
        shots = {
            "normal": session(now, percent=41, total_cost=185.4, minutes=5472, api_minutes=328, output_tokens=1000, cache_read=82_000, input_tokens=800),
            "danger": session(now, percent=88, total_cost=204.6, minutes=7117, api_minutes=427, output_tokens=1000, cache_read=176_000, input_tokens=1800),
        }
        args.out.mkdir(parents=True, exist_ok=True)
        for name, payload in shots.items():
            payload.update(common)
            lines = run_howl(args.howl, home, payload)
            render(lines, args.fonts).save(args.out / f"{name}.png")
            print(f"{name}: {args.out / f'{name}.png'}")
            for line in lines:
                print("  " + plain(line))


if __name__ == "__main__":
    main()
