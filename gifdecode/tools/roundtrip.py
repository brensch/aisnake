"""Side-by-side round trip: original GIF frame (left) | frame redrawn from decoded state (right).

    python3 roundtrip.py <game.gif> <decoded game.json> <out.mp4>

The decoded game is one line of snakedecode's output. Needs Pillow and ffmpeg on PATH.
"""
import json
import subprocess
import sys

from PIL import Image, ImageDraw

FF = "ffmpeg"
S, PITCH, ORIGIN, N = 2, 20, 3, 11  # draw at 2x


def centre(x, y):  # Battlesnake coords -> pixel centre at 2x
    row = N - 1 - y
    return ((ORIGIN + PITCH * x + 9) * S, (ORIGIN + PITCH * row + 9) * S)


def draw_state(turn, colours):
    im = Image.new("RGB", (224 * S, 224 * S), (255, 255, 255))
    d = ImageDraw.Draw(im)
    for i in range(N):
        for j in range(N):
            x0, y0 = (ORIGIN + PITCH * i) * S, (ORIGIN + PITCH * j) * S
            d.rectangle([x0, y0, x0 + 18 * S - 1, y0 + 18 * S - 1], fill=(240, 240, 240))
    for fx, fy in turn["food"]:
        cx, cy = centre(fx, fy)
        d.ellipse([cx - 6 * S, cy - 6 * S, cx + 6 * S, cy + 6 * S], fill=(255, 92, 117))
    for body, col in zip(turn["snakes"], colours):
        if not body:
            continue
        pts = [centre(x, y) for x, y in body]
        if len(pts) > 1:
            d.line(pts, fill=col, width=18 * S, joint="curve")
        for px, py in pts:
            d.rounded_rectangle([px - 9 * S, py - 9 * S, px + 9 * S, py + 9 * S], radius=6 * S, fill=col)
        hx, hy = pts[0]  # eyes on the head so direction is visible
        d.ellipse([hx - 4 * S, hy - 4 * S, hx + 4 * S, hy + 4 * S], fill=(255, 255, 255))
        d.ellipse([hx - 2 * S, hy - 2 * S, hx + 2 * S, hy + 2 * S], fill=(0, 0, 0))
        d.text((hx + 6 * S, hy - 9 * S), str(len(body)), fill=(0, 0, 0))
    return im


def main(gif, decoded, out):
    g = json.load(open(decoded))
    src = Image.open(gif)
    W, H = 224 * S * 2 + 8, 224 * S + 24
    proc = subprocess.Popen([FF, "-loglevel", "error", "-y", "-f", "rawvideo", "-pix_fmt", "rgb24",
                             "-s", f"{W}x{H}", "-r", "12.5", "-i", "-", "-pix_fmt", "yuv420p",
                             "-c:v", "libx264", "-crf", "20", out], stdin=subprocess.PIPE)
    for t, turn in enumerate(g["turns"]):
        src.seek(t)
        left = src.convert("RGB").resize((224 * S, 224 * S), Image.NEAREST)
        frame = Image.new("RGB", (W, H), (20, 20, 20))
        frame.paste(left, (0, 24))
        frame.paste(draw_state(turn, g["colors"]), (224 * S + 8, 24))
        dd = ImageDraw.Draw(frame)
        dd.text((6, 6), f"original GIF   turn {t}", fill=(255, 255, 255))
        dd.text((224 * S + 14, 6), "redrawn from decoded game state", fill=(255, 255, 255))
        proc.stdin.write(frame.tobytes())
    proc.stdin.close()
    proc.wait()


if __name__ == "__main__":
    main(*sys.argv[1:4])
