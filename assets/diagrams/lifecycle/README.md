# ClassGame content lifecycle diagram

The slide-ready loop that a single mission travels: teacher intent becomes a
draft, gets published, is played by the class, returns as evidence, and comes
back to the teacher as insight.

## Files

- `content_lifecycle_graphviz.py` — self-contained Graphviz renderer with
  inline `uv` dependencies.
- `content-lifecycle.png` — generated 4K (3840×2160) image.
- `content-lifecycle.svg` — self-contained 1920×1080 vector version.

## Regenerate

From this directory:

```bash
uv run content_lifecycle_graphviz.py
```

## How it is laid out

It reuses the slide language of `../architecture/architecture.py`:
the same palette, the same icon cards (icon on top, bold title, muted
subtitle), the same quiet rounded panels, and the same 16:9 canvas with the
render fitted under the header band.

The engine is `neato` rather than `dot`, because the ring is a fixed geometry:
every card is pinned on an ellipse (`RING_RX`/`RING_RY`) and Graphviz only
routes the edges. Two consequences worth knowing before editing:

- `inputscale=72` is required, or neato reads the point positions as inches.
- neato has no clusters, so each panel is a pinned background node sized to the
  cards it sits behind, drawn first with `outputorder=nodesfirst` so the edges
  paint on top.

## What the diagram represents

Six stages run clockwise from the left, grouped into the panels that own them —
`TEACHER`, `AUTHORING`, `CLASSROOM`, `LEARNING` — with `AGENT RUNTIME` in the
middle. The dashed spokes name which of M.I.A's agents runs a stage: the
Mission Agent drafts, the Tutor Agent works alongside the class, and the
Learning Agent turns evidence into insight.
