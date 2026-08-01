# Adaptive poster compositor: backend handoff

The release poster is composed deterministically in the frontend. The GPU provides a clean, text-free key visual plus layout evidence; titles, factual copy, original logos and QR codes must not be redrawn by diffusion.

## Required candidate response

Add this optional object to every candidate `spec`. Existing clients remain compatible when it is absent.

```json
{
  "visualAnalysis": {
    "palette": ["#17121f", "#94ad48", "#d1bd62"],
    "contrast": "high",
    "texture": "hand_drawn_grain",
    "typographyProfile": {
      "family": "expressive_serif",
      "weight": 900,
      "fillColor": "#090909",
      "strokeColor": "#d1bd62",
      "distress": 0.45
    },
    "subjectBounds": [{"x": 0.22, "y": 0.12, "w": 0.62, "h": 0.68}],
    "textSafeZones": [
      {"x": 0.04, "y": 0.05, "w": 0.30, "h": 0.46, "align": "left", "priority": 1}
    ],
    "logoSafeZone": {"x": 0.06, "y": 0.72, "w": 0.88, "h": 0.12},
    "hasGeneratedText": false,
    "ocrDetections": []
  }
}
```

Coordinates are normalized to the image. `subjectBounds` includes faces, people, buildings, venues, products and other focal objects. Safe zones must not intersect those bounds.

## Generation and validation requirements

1. Generate a clean background without words, letters, numbers, logos, signatures, watermarks or ticket information.
2. Analyse palette, luminance, contrast, texture and composition after generation, not only from the prompt.
3. Return two to four ranked text-safe zones and one logo-safe zone.
4. Run OCR before marking a candidate ready. If generated text is detected, retry or inpaint it away.
5. Never recreate an uploaded logo through diffusion. Preserve the original asset for deterministic composition.
6. Keep the raw candidate endpoint available after finalization. The frontend release compositor uses the selected raw candidate, not an already typeset `/result` image.
7. Return an explicit analysis error instead of inventing safe zones when analysis fails.

## Acceptance criteria

- `hasGeneratedText=false` and `ocrDetections=[]` for a publishable candidate.
- Every safe zone is inside the image and does not overlap a subject bound.
- Palette contains valid sRGB hex colors.
- The raw candidate remains downloadable at 1024 × 1536.
- Original logo identity is pixel-preserved.
