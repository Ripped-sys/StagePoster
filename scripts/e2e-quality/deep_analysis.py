from PIL import Image
import numpy as np
from collections import Counter
import sys, os, json

# ── paths ────────────────────────────────────────────────────────────────────
PATHS = {
    "poster": "/workspace/poster-engine/scripts/e2e-quality/poster_poster_56679f97-b470-4988-8fbf-b8fe060ee5de.png",
    "candidate": "/workspace/poster-engine/scripts/e2e-quality/candidate_candidate_e51db786-92a1-44a8-8c49-59721cfb2c94.png",
    "thumb": "/workspace/poster-engine/scripts/e2e-quality/thumb_poster_56679f97-b470-4988-8fbf-b8fe060ee5de.png",
}

for k, p in PATHS.items():
    if not os.path.exists(p):
        print(f"ERROR: {k}: {p} not found")
        sys.exit(1)

results = {}


# ── load helper ─────────────────────────────────────────────────────────────
def load(name):
    im = Image.open(PATHS[name]).convert("RGB")
    arr = np.array(im, dtype=np.float32) / 255.0
    return im, arr


# ── 1. pixel statistics ─────────────────────────────────────────────────────
def pixel_stats(arr):
    h, w = arr.shape[:2]
    total = h * w
    r, g, b = arr[:, :, 0], arr[:, :, 1], arr[:, :, 2]
    return {
        "shape": (h, w),
        "mean_r": round(float(r.mean()), 4),
        "mean_g": round(float(g.mean()), 4),
        "mean_b": round(float(b.mean()), 4),
        "std_r": round(float(r.std()), 4),
        "std_g": round(float(g.std()), 4),
        "std_b": round(float(b.std()), 4),
        "min_r": round(float(r.min()), 4),
        "min_g": round(float(g.min()), 4),
        "min_b": round(float(b.min()), 4),
        "max_r": round(float(r.max()), 4),
        "max_g": round(float(g.max()), 4),
        "max_b": round(float(b.max()), 4),
        "overall_mean": round(float(arr.mean()), 4),
        "overall_std": round(float(arr.std()), 4),
    }


# ── 2. color histogram (top-10) ─────────────────────────────────────────────
def top_colors(arr, n=10):
    flat = arr.reshape(-1, 3)
    # Quantize to 16 levels per channel → 4096 bins
    q = (flat * 15).astype(np.uint8)
    # build compact key as tuple
    counts = Counter([tuple(p) for p in q])
    total = len(flat)
    top = counts.most_common(n)
    return [
        {"color": list(c), "count": v, "pct": round(v / total * 100, 3)} for c, v in top
    ]


# ── 3 & 4. band analysis ────────────────────────────────────────────────────
def band_stats(arr, name):
    h = arr.shape[0]
    if name == "top":
        b = arr[: int(h * 0.20)]
    elif name == "mid":
        b = arr[int(h * 0.20) : int(h * 0.78)]
    else:
        b = arr[int(h * 0.78) :]

    if b.size == 0:
        return {"band": name, "mean": 0, "std": 0, "dominant": []}

    mean = float(b.mean())
    std  = float(b.std())
    grey1 = (np.abs(b[:,:,0] - b[:,:,1]) < 0.05)
    grey2 = (np.abs(b[:,:,1] - b[:,:,2]) < 0.05)
    grey_ratio = float((grey1 & grey2).mean())
    hist = top_colors(b, 5)
    return {
        "band": name,
        "mean": round(mean, 4),
        "std": round(std, 4),
        "grey_ratio": round(grey_ratio, 4),
        "dominant": hist,
    }


# ── 5. edge density (Sobel) ─────────────────────────────────────────────────
def edge_density(arr, name):
    h = arr.shape[0]
    if name == "top":
        b = arr[: int(h * 0.20)]
    elif name == "mid":
        b = arr[int(h * 0.20) : int(h * 0.78)]
    else:
        b = arr[int(h * 0.78) :]

    gray = b.mean(axis=2)
    gy = np.abs(np.diff(gray, axis=0))  # vertical gradient
    gx = np.abs(np.diff(gray, axis=1))  # horizontal gradient
    # tile gy to match gx on the inner region
    edges = gy[:, :-1] + gx[:-1, :]
    d = float(edges.mean())
    return round(d, 6)


# ── 6. color difference poster vs candidate ─────────────────────────────────
def color_diff(arr_a, arr_b):
    # resize candidate to match poster if needed
    ha, wa = arr_a.shape[:2]
    hb, wb = arr_b.shape[:2]
    if (ha, wa) != (hb, wb):
        img = Image.fromarray((arr_b * 255).astype(np.uint8))
        img = img.resize((wa, ha), Image.BILINEAR)
        arr_b = np.array(img, dtype=np.float32) / 255.0
    diff = np.abs(arr_a - arr_b)
    return {
        "mean_abs_diff": round(float(diff.mean()), 6),
        "max_abs_diff": round(float(diff.max()), 6),
        "mean_r_diff": round(float(diff[:, :, 0].mean()), 6),
        "mean_g_diff": round(float(diff[:, :, 1].mean()), 6),
        "mean_b_diff": round(float(diff[:, :, 2].mean()), 6),
    }


# ── 7. large white / light rectangles ───────────────────────────────────────
def white_panels(arr, threshold=0.90, min_area_frac=0.03):
    """Detect large contiguous light rectangles (text overlay panels)."""
    h, w = arr.shape[:2]
    # brightness
    bright = arr.mean(axis=2) > threshold
    # find connected-ish regions: simple grid scan → bounding boxes
    #   use run-length on rows to find horizontal spans of bright pixels
    panels = []
    for y in range(h):
        in_run = False
        run_start = 0
        for x in range(w):
            if bright[y, x]:
                if not in_run:
                    in_run = True
                    run_start = x
            else:
                if in_run:
                    span = (run_start, x)
                    # check if this span extends vertically
                    panels.append((y, run_start, x))
                    in_run = False
        if in_run:
            panels.append((y, run_start, w))

    # collapse rows into vertical spans
    from collections import defaultdict

    cols = defaultdict(list)
    for y, xs, xe in panels:
        cols[(xs, xe)].append(y)

    min_rows = int(h * min_area_frac)
    found = []
    for (xs, xe), rows in cols.items():
        if len(rows) < min_rows:
            continue
        y0, y1 = min(rows), max(rows)
        area_frac = (y1 - y0) * (xe - xs) / (h * w)
        avg_bright = arr[y0:y1, xs:xe].mean()
        if area_frac >= min_area_frac:
            found.append(
                {
                    "y0": int(y0),
                    "y1": int(y1),
                    "x0": int(xs),
                    "x1": int(xe),
                    "area_frac": round(area_frac, 4),
                    "avg_brightness": round(float(avg_bright), 4),
                }
            )
    return found


# ── helper: img info ────────────────────────────────────────────────────────
def img_info(name):
    im = Image.open(PATHS[name])
    return {
        "filename": os.path.basename(PATHS[name]),
        "format": im.format,
        "mode": im.mode,
        "size": im.size,
        "file_size_kb": os.path.getsize(PATHS[name]) // 1024,
    }


# ════════════════════════════════════════════════════════════════════════════
# RUN
# ════════════════════════════════════════════════════════════════════════════

report_lines = []


def banner(t):
    report_lines.append("=" * 70)
    report_lines.append(f"  {t}")
    report_lines.append("=" * 70)


def section(t):
    report_lines.append(f"\n── {t} ──")


# ── file info ───────────────────────────────────────────────────────────────
banner("IMAGE FILE INFO")
for k in PATHS:
    info = img_info(k)
    report_lines.append(f"\n[{k}]")
    for kk, vv in info.items():
        report_lines.append(f"  {kk}: {vv}")

# ── load all ────────────────────────────────────────────────────────────────
im_poster, arr_poster = load("poster")
im_cand, arr_cand = load("candidate")
im_thumb, arr_thumb = load("thumb")

# ── 1. pixel stats ─────────────────────────────────────────────────────────
banner("PIXEL STATISTICS")
for name, arr in [
    ("poster", arr_poster),
    ("candidate", arr_cand),
    ("thumb", arr_thumb),
]:
    s = pixel_stats(arr)
    section(f"{name}")
    report_lines.append(f"  shape     : {s['shape']}")
    report_lines.append(
        f"  overall   : mean={s['overall_mean']}  std={s['overall_std']}"
    )
    report_lines.append(
        f"  R         : mean={s['mean_r']}  std={s['std_r']}  min={s['min_r']}  max={s['max_r']}"
    )
    report_lines.append(
        f"  G         : mean={s['mean_g']}  std={s['std_g']}  min={s['min_g']}  max={s['max_g']}"
    )
    report_lines.append(
        f"  B         : mean={s['mean_b']}  std={s['std_b']}  min={s['min_b']}  max={s['max_b']}"
    )

# ── 2. top colors ──────────────────────────────────────────────────────────
banner("TOP 10 COLORS (16-level quantized)")
for name, arr in [
    ("poster", arr_poster),
    ("candidate", arr_cand),
    ("thumb", arr_thumb),
]:
    tc = top_colors(arr, 10)
    section(f"{name}")
    for i, item in enumerate(tc):
        rgb = item["color"]
        report_lines.append(
            f"  #{i + 1:02d}  RGB({rgb[0] * 17:3d},{rgb[1] * 17:3d},{rgb[2] * 17:3d})  "
            f"count={item['count']:>8}  {item['pct']:.2f}%"
        )

# ── 3 & 4 & 5. band analysis ───────────────────────────────────────────────
banner("BAND ANALYSIS (top 0-20% | mid 20-78% | bot 78-100%)")
for name, arr in [("poster", arr_poster), ("candidate", arr_cand)]:
    section(f"{name}")
    for bn in ["top", "mid", "bot"]:
        bs = band_stats(arr, bn)
        ed = edge_density(arr, bn)
        labels = {
            "top": "Top  (y=0–20%)",
            "mid": "Mid  (y=20–78%)",
            "bot": "Bot  (y=78–100%)",
        }
        report_lines.append(f"\n  [{labels[bn]}]")
        report_lines.append(f"    mean brightness : {bs['mean']}")
        report_lines.append(f"    contrast (std)  : {bs['std']}")
        report_lines.append(
            f"    grey_ratio      : {bs['grey_ratio']}  (>0.7=very grey)"
        )
        report_lines.append(f"    edge density    : {ed}")
        if bs["dominant"]:
            d0 = bs["dominant"][0]
            report_lines.append(
                f"    dominant color  : RGB({d0['color'][0] * 17},"
                f"{d0['color'][1] * 17},{d0['color'][2] * 17}) "
                f"({d0['pct']:.1f}%)"
            )

# ── 6. color diff ──────────────────────────────────────────────────────────
banner("COLOR DIFFERENCE: poster vs candidate")
diff = color_diff(arr_poster, arr_cand)
for kk, vv in diff.items():
    report_lines.append(f"  {kk}: {vv}")

# ── 7. white panels ─────────────────────────────────────────────────────────
banner("LARGE LIGHT PANELS (white/light rectangles)")
for name, arr in [("poster", arr_poster), ("candidate", arr_cand)]:
    panels = white_panels(arr, threshold=0.88, min_area_frac=0.02)
    section(name)
    if not panels:
        report_lines.append("  No large light panels detected.")
    else:
        for p in panels:
            report_lines.append(
                f"  y={p['y0']:4d}–{p['y1']:4d}  "
                f"x={p['x0']:4d}–{p['x1']:4d}  "
                f"area={p['area_frac']:.1%}  "
                f"brightness={p['avg_brightness']}"
            )

# ── 8. blankness check ─────────────────────────────────────────────────────
banner("BLANKNESS / UNIFORMITY DIAGNOSTIC")
h_p, w_p = arr_poster.shape[:2]

# check if poster is near-uniform
poster_std = arr_poster.std()
report_lines.append(f"  Poster overall std: {poster_std:.6f}")
if poster_std < 0.05:
    report_lines.append("  ⚠  VERY LOW STD (< 0.05) — poster may be nearly blank")
elif poster_std < 0.10:
    report_lines.append("  ⚠  LOW STD (< 0.10) — poster may be unusually uniform")
else:
    report_lines.append("  ✓  Std is in normal range for a photograph/AI image")

# check if poster is predominantly one color
flat = arr_poster.reshape(-1, 3)
q = (flat * 15).astype(np.uint8)
counts = Counter([tuple(p) for p in q])
total = len(flat)
top1_pct = counts.most_common(1)[0][1] / total
report_lines.append(f"  Top-1 color covers {top1_pct:.1%} of pixels")
if top1_pct > 0.90:
    report_lines.append("  ⚠  Extremely uniform — likely blank or single-color fill")
elif top1_pct > 0.80:
    report_lines.append("  ⚠  Highly uniform — possible blank area or gradient fill")
else:
    report_lines.append("  ✓  Good color diversity")

# per-band std for poster
for bn in ["top", "mid", "bot"]:
    h = arr_poster.shape[0]
    if bn == "top":
        b = arr_poster[: int(h * 0.20)]
    elif bn == "mid":
        b = arr_poster[int(h * 0.20) : int(h * 0.78)]
    else:
        b = arr_poster[int(h * 0.78) :]
    report_lines.append(f"  Band {bn} std: {b.std():.6f}")

report_lines.append("")
banner("END OF REPORT")

# ── write ───────────────────────────────────────────────────────────────────
out = "/workspace/poster-engine/scripts/e2e-quality/deep-analysis.txt"
with open(out, "w") as f:
    f.write("\n".join(report_lines) + "\n")
print(f"Report written to {out}")
print("\n".join(report_lines))
