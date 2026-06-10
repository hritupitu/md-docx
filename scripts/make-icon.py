#!/usr/bin/env python3
"""
Generates md2docx.icns — the macOS app icon.

Design: dark rounded square, stylised document with a markdown
        downward-arrow mark, cyan accent.
"""

import math, os, shutil, struct, subprocess, sys, zlib
from pathlib import Path

try:
    from PIL import Image, ImageDraw, ImageFilter
except ImportError:
    subprocess.check_call([sys.executable, "-m", "pip", "install", "Pillow", "-q"])
    from PIL import Image, ImageDraw, ImageFilter

# ── palette ───────────────────────────────────────────────────────────────────
BG      = (10,  14,  20)      # near-black navy
DOC     = (235, 238, 242)     # paper white
DOC_DIM = (195, 200, 210)     # fold shadow
ACCENT  = (0,   212, 255)     # cyan
DARK    = (6,    9,  14)      # deeper shadow

SIZE = 1024

# ── helpers ───────────────────────────────────────────────────────────────────

def rounded_rect_mask(size, radius):
    """Return an L-mode mask image with a rounded rectangle."""
    mask = Image.new("L", (size, size), 0)
    d = ImageDraw.Draw(mask)
    d.rounded_rectangle([0, 0, size - 1, size - 1], radius=radius, fill=255)
    return mask


def draw_arrow_down(draw, cx, cy, w, h, color, thickness):
    """Draw a bold downward-pointing block arrow centred at (cx, cy)."""
    hw = w // 2
    shaft_w = int(w * 0.36)
    shaft_h = int(h * 0.52)
    head_h  = h - shaft_h

    # shaft
    draw.rectangle(
        [cx - shaft_w // 2, cy - h // 2,
         cx + shaft_w // 2, cy - h // 2 + shaft_h],
        fill=color,
    )
    # arrowhead
    draw.polygon(
        [
            (cx - hw,       cy - h // 2 + shaft_h),
            (cx + hw,       cy - h // 2 + shaft_h),
            (cx,            cy + h // 2),
        ],
        fill=color,
    )


def make_icon(size=SIZE):
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    s = size / SIZE  # scale factor

    # ── background ────────────────────────────────────────────────────────────
    radius = int(224 * s)
    mask = rounded_rect_mask(size, radius)
    bg = Image.new("RGBA", (size, size), (*BG, 255))
    img.paste(bg, mask=mask)

    # subtle inner gradient overlay (darker at top, lighter at bottom)
    grad = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    gd = ImageDraw.Draw(grad)
    for y in range(size):
        alpha = int(40 * (1 - y / size))
        gd.line([(0, y), (size, y)], fill=(255, 255, 255, alpha))
    img = Image.alpha_composite(img, grad)
    draw = ImageDraw.Draw(img)

    # ── document shape ────────────────────────────────────────────────────────
    # left-aligned, tall portrait document
    doc_l = int(170 * s)
    doc_t = int(145 * s)
    doc_r = int(590 * s)
    doc_b = int(880 * s)
    fold  = int(115 * s)

    # shadow (soft, offset down-right)
    shadow_offset = int(18 * s)
    shadow = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    sd = ImageDraw.Draw(shadow)
    sd.polygon([
        (doc_l + shadow_offset, doc_t + shadow_offset),
        (doc_r - fold + shadow_offset, doc_t + shadow_offset),
        (doc_r + shadow_offset, doc_t + fold + shadow_offset),
        (doc_r + shadow_offset, doc_b + shadow_offset),
        (doc_l + shadow_offset, doc_b + shadow_offset),
    ], fill=(0, 0, 0, 90))
    shadow = shadow.filter(ImageFilter.GaussianBlur(radius=int(22 * s)))
    img = Image.alpha_composite(img, shadow)
    draw = ImageDraw.Draw(img)

    # document body
    draw.polygon([
        (doc_l, doc_t),
        (doc_r - fold, doc_t),
        (doc_r, doc_t + fold),
        (doc_r, doc_b),
        (doc_l, doc_b),
    ], fill=(*DOC, 255))

    # fold triangle
    draw.polygon([
        (doc_r - fold, doc_t),
        (doc_r,        doc_t + fold),
        (doc_r - fold, doc_t + fold),
    ], fill=(*DOC_DIM, 255))

    # thin fold crease line
    draw.line(
        [(doc_r - fold, doc_t), (doc_r - fold, doc_t + fold)],
        fill=(*DOC_DIM, 200), width=max(1, int(2 * s))
    )

    # ── markdown logo on document ─────────────────────────────────────────────
    # Classic markdown badge: bold "M" + down-arrow block — placed in the
    # lower-centre of the document area.

    # "M" — drawn as two vertical pillars + two diagonals
    def draw_M(draw, lx, ty, w, h, color, thick):
        draw.rectangle([lx,          ty, lx + thick,       ty + h], fill=color)
        draw.rectangle([lx + w - thick, ty, lx + w,        ty + h], fill=color)
        mid_x = lx + w // 2
        mid_y = ty + int(h * 0.48)
        draw.polygon([
            (lx,            ty),
            (lx + thick * 2, ty),
            (mid_x,         mid_y),
            (lx + w - thick * 2, ty),
            (lx + w,        ty),
            (mid_x,         mid_y + int(h * 0.08)),
        ], fill=color)

    # position: centred horizontally in doc, lower half
    m_w  = int(260 * s)
    m_h  = int(180 * s)
    m_tk = int(38 * s)
    m_cx = (doc_l + doc_r) // 2
    m_ty = int(430 * s)
    draw_M(draw, m_cx - m_w // 2, m_ty, m_w, m_h, DARK, m_tk)

    # downward arrow — below the M, same width
    arr_w = int(240 * s)
    arr_h = int(150 * s)
    arr_cx = m_cx
    arr_cy = m_ty + m_h + int(20 * s) + arr_h // 2
    draw_arrow_down(draw, arr_cx, arr_cy, arr_w, arr_h, DARK, 0)

    # ── cyan accent lines on document (like text lines) ───────────────────────
    line_x0 = doc_l + int(55 * s)
    line_x1 = doc_r - int(55 * s)
    line_w   = max(2, int(6 * s))
    line_alpha = 160

    for i, y_frac in enumerate([0.175, 0.225, 0.275]):
        ly = doc_t + int((doc_b - doc_t) * y_frac)
        x1 = line_x1 if i > 0 else (line_x0 + int((line_x1 - line_x0) * 0.6))
        draw.rectangle([line_x0, ly, x1, ly + line_w],
                       fill=(*ACCENT, line_alpha))

    # ── cyan "docx" badge (bottom-right corner of canvas) ────────────────────
    badge_r  = int(110 * s)          # corner radius
    badge_pad = int(28 * s)
    bw, bh   = int(290 * s), int(110 * s)
    bx = size - bw - int(52 * s)
    by = size - bh - int(52 * s)

    # badge background
    bdg = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    bd  = ImageDraw.Draw(bdg)
    bd.rounded_rectangle([bx, by, bx + bw, by + bh],
                          radius=int(30 * s), fill=(*ACCENT, 245))
    img = Image.alpha_composite(img, bdg)
    draw = ImageDraw.Draw(img)

    # ".docx" text rendered as simple pixel rectangles (avoids font headaches)
    # Instead, draw bold label using a series of thin rectangles spelling "DOCX"
    # — but that's very hard without a font. Use a tiny repeating-rect trick or
    #   just render "DOCX" using Pillow's built-in bitmap font.
    try:
        from PIL import ImageFont
        # try a bold system font
        for font_path in [
            "/System/Library/Fonts/Supplemental/Arial Bold.ttf",
            "/System/Library/Fonts/Helvetica.ttc",
            "/System/Library/Fonts/SFNSMono.ttf",
            "/System/Library/Fonts/SFCompact.ttf",
        ]:
            if os.path.exists(font_path):
                font = ImageFont.truetype(font_path, size=int(62 * s))
                break
        else:
            font = ImageFont.load_default()

        text = ".docx"
        bbox = font.getbbox(text)
        tw, th = bbox[2] - bbox[0], bbox[3] - bbox[1]
        tx = bx + (bw - tw) // 2 - bbox[0]
        ty2 = by + (bh - th) // 2 - bbox[1]
        draw.text((tx, ty2), text, font=font, fill=(*DARK, 255))
    except Exception:
        pass   # badge stays blank — still looks like an accent pill

    # ── apply rounded-square clip ─────────────────────────────────────────────
    mask = rounded_rect_mask(size, radius)
    img.putalpha(mask)

    return img


# ── build iconset ─────────────────────────────────────────────────────────────

def build_icns(out_icns: Path):
    iconset = out_icns.with_suffix(".iconset")
    iconset.mkdir(parents=True, exist_ok=True)

    master = make_icon(SIZE)

    sizes = [16, 32, 64, 128, 256, 512, 1024]
    for sz in sizes:
        img = master.resize((sz, sz), Image.LANCZOS)
        img.save(iconset / f"icon_{sz}x{sz}.png")
        if sz <= 512:
            img2 = master.resize((sz * 2, sz * 2), Image.LANCZOS)
            img2.save(iconset / f"icon_{sz}x{sz}@2x.png")

    subprocess.check_call(["iconutil", "-c", "icns", str(iconset), "-o", str(out_icns)])
    shutil.rmtree(iconset)
    print(f"✔  Icon written to {out_icns}  ({out_icns.stat().st_size // 1024} KB)")


if __name__ == "__main__":
    out = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("build/md2docx.icns")
    out.parent.mkdir(parents=True, exist_ok=True)
    build_icns(out)
