# /// script
# requires-python = ">=3.11"
# dependencies = ["diagrams>=0.25,<0.26", "pillow>=11,<13"]
# ///
"""Render the ClassGame content lifecycle as an icon-led loop.

Same slide language as `../architecture/architecture.py` — compact
icon cards (icon on top, bold title, muted subtitle), the same palette, and a
render fitted below a slide header on an exact 16:9 canvas — applied to the
circular lifecycle shape. Six stages sit pinned on an ellipse so the loop reads
clockwise from the teacher's intent, with M.I.A in the middle wired to the
three stages its agents own.

Graphviz is driven by `neato` here rather than `dot`: the ring is a fixed
geometry, so the positions are pinned and the engine only routes the edges.

Run from any directory:

    uv run content_lifecycle_graphviz.py

Outputs beside this file:

- content-lifecycle.png — 4K raster image.
- content-lifecycle.svg — self-contained vector image.
"""

import base64
import math
import mimetypes
from pathlib import Path
import xml.etree.ElementTree as ET

from diagrams import Diagram, Edge, Node
from diagrams.gcp.database import Firestore
from diagrams.gcp.ml import AIPlatform, VertexAI
from diagrams.gcp.operations import Monitoring
from diagrams.gcp.storage import GCS
from diagrams.onprem.client import Users
from diagrams.programming.framework import React
from PIL import Image, ImageDraw, ImageFont, ImageOps


ROOT = Path(__file__).resolve().parent
RAW_OUTPUT = ROOT / "content-lifecycle-raw"
OUTPUT = ROOT / "content-lifecycle.png"
SVG_OUTPUT = ROOT / "content-lifecycle.svg"

LOGICAL_SIZE = (1920, 1080)
PNG_SCALE = 2
CANVAS_SIZE = tuple(dimension * PNG_SCALE for dimension in LOGICAL_SIZE)
# Slide header band (logical px). The graph is fitted into the space below it.
HEADER_HEIGHT = 112
FRAME_MARGIN_X = 48
FRAME_MARGIN_BOTTOM = 40

TITLE = "ClassGame · Content Lifecycle"
SUBTITLE = "One loop per mission: teacher intent · M.I.A draft · class play · evidence · insight"

BLUE = "#3b63f3"
PURPLE = "#8b5cf6"
ORANGE = "#ed8b00"
TEAL = "#0ea5b7"
GREEN = "#16a368"
INK = "#211833"
MUTED = "#645b73"
SPOKE = "#c3c8dc"
PANEL_FILL = "#f6f7fb"
PANEL_LINE = "#d8dce8"
SANS = "Helvetica Neue, Helvetica, Arial, sans-serif"

# Ring geometry in Graphviz points (y grows upwards). Wider than tall so the
# finished loop fills the 16:9 slide instead of letterboxing into a square.
RING_RX = 760
RING_RY = 400
# Clockwise on screen from the left, so the loop starts at the teacher and
# travels over the top before coming back along the bottom.
STAGE_ANGLES = (180, 240, 300, 0, 60, 120)
# Card footprint in points. The panels are measured from it, so `card()`
# derives its own width and height here too.
CARD_SIZE = (104.0, 144.0)
PANEL_PAD = 58.0


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    """Load the same title fonts used by the architecture slide."""
    candidates = [
        "/System/Library/Fonts/Supplemental/Arial Bold.ttf"
        if bold
        else "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/System/Library/Fonts/Supplemental/Helvetica Bold.ttf"
        if bold
        else "/System/Library/Fonts/Supplemental/Helvetica.ttf",
        "/Library/Fonts/Arial Bold.ttf" if bold else "/Library/Fonts/Arial.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
        if bold
        else "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
    ]
    for candidate in candidates:
        if Path(candidate).exists():
            return ImageFont.truetype(candidate, size=size)
    return ImageFont.load_default()


FONT_TITLE = font(36 * PNG_SCALE, bold=True)
FONT_SUBTITLE = font(17 * PNG_SCALE)

# Positions are authored, not searched: `overlap` stays on so neato leaves the
# pinned ring alone and only routes splines around the cards.
graph_attr = {
    "bgcolor": "white",
    "dpi": "192",
    "pad": "0.8",
    # `pos` below is written in points; without this neato would read it as
    # inches and blow the canvas up 72x.
    "inputscale": "72",
    "splines": "spline",
    "overlap": "true",
    "outputorder": "nodesfirst",
    "fontname": "Helvetica",
}
node_attr = {
    "fontname": "Helvetica",
    "fontcolor": INK,
}
edge_attr = {
    "color": BLUE,
    "fontcolor": MUTED,
    "fontname": "Helvetica",
    "fontsize": "12.5",
    "penwidth": "1.8",
    "arrowsize": "0.8",
}


def ring_point(angle_deg: float) -> tuple[float, float]:
    """A spot on the ellipse. Screen angles, so the y axis is flipped."""
    radians = math.radians(angle_deg)
    return RING_RX * math.cos(radians), -RING_RY * math.sin(radians)


def pin(point: tuple[float, float]) -> str:
    return f"{point[0]:.1f},{point[1]:.1f}!"


def panel(title: str, *points: tuple[float, float]) -> Node:
    """A quiet rounded panel behind one or more cards.

    neato has no clusters, so the architecture's panel is reproduced as a
    pinned, unconnected background node sized to the cards it sits behind. The
    title is an HTML label stretched to the panel width, which is the only way
    to right-align it the way a real cluster label sits.
    """
    half_width, half_height = CARD_SIZE[0] / 2, CARD_SIZE[1] / 2
    xs = [point[0] for point in points]
    ys = [point[1] for point in points]
    left, right = min(xs) - half_width - PANEL_PAD, max(xs) + half_width + PANEL_PAD
    bottom, top = min(ys) - half_height - PANEL_PAD, max(ys) + half_height + PANEL_PAD
    # The width belongs on the cell, not the table: a table only stretches its
    # own frame, so the text would stay pinned to the left.
    label = (
        f'<<table border="0" cellborder="0" cellspacing="0">'
        f'<tr><td width="{right - left - 40:.0f}" align="right">'
        f'<font point-size="13" color="{MUTED}"><b>{title}</b></font>'
        f"</td></tr></table>>"
    )
    return Node(
        label=label,
        shape="box",
        style="rounded,filled",
        fillcolor=PANEL_FILL,
        color=PANEL_LINE,
        penwidth="1.2",
        fontname="Helvetica-Bold",
        labelloc="t",
        fixedsize="true",
        width=f"{(right - left) / 72:.3f}",
        height=f"{(top - bottom) / 72:.3f}",
        pos=pin(((left + right) / 2, (top + bottom) / 2)),
    )


def card(icon: type[Node], title: str, subtitle: str, **attrs) -> Node:
    """Icon on top, bold title, muted subtitle — one compact card per stage."""
    label = (
        f'<<font point-size="16"><b>{title}</b></font>'
        f'<br/><font point-size="12" color="{MUTED}">{subtitle}</font>>'
    )
    return icon(
        label,
        width=f"{CARD_SIZE[0] / 72:.3f}",
        height=f"{CARD_SIZE[1] / 72:.3f}",
        fixedsize="true",
        imagescale="true",
        imagepos="tc",
        labelloc="b",
        **attrs,
    )


def step(color: str, label: str) -> Edge:
    """A hop along the ring."""
    return Edge(color=color, label=label)


def spoke(agent: str) -> Edge:
    """Which of M.I.A's agents owns a stage. Named, but never louder than the loop."""
    return Edge(
        color=SPOKE,
        label=agent,
        style="dashed",
        penwidth="1.2",
        arrowhead="none",
        fontsize="11.5",
    )


with Diagram(
    "",
    filename=str(RAW_OUTPUT),
    show=False,
    outformat=["png", "svg"],
    graph_attr=graph_attr,
    node_attr=node_attr,
    edge_attr=edge_attr,
) as diagram:
    diagram.dot.engine = "neato"

    stage_points = [ring_point(angle) for angle in STAGE_ANGLES]
    panel("TEACHER", stage_points[0])
    panel("AUTHORING", stage_points[1], stage_points[2])
    panel("CLASSROOM", stage_points[3])
    panel("LEARNING", stage_points[4], stage_points[5])
    panel("AGENT RUNTIME", (0.0, 0.0))

    brief = card(
        Users,
        "Teacher intent",
        "topic · grade · goal",
        pos=pin(stage_points[0]),
    )
    draft = card(
        AIPlatform,
        "Mission draft",
        "grounded draft · JSON",
        pos=pin(stage_points[1]),
    )
    publish = card(
        GCS,
        "Reviewed + published",
        "approved · immutable copy",
        pos=pin(stage_points[2]),
    )
    play = card(
        React,
        "Class plays",
        "3D mission · Mia tutors",
        pos=pin(stage_points[3]),
    )
    evidence = card(
        Firestore,
        "Evidence",
        "attempts · turns · live",
        pos=pin(stage_points[4]),
    )
    insight = card(
        Monitoring,
        "Class insight",
        "gaps · mastery · next step",
        pos=pin(stage_points[5]),
    )
    mia = card(
        VertexAI,
        "M.I.A",
        "Gemini · class-grounded",
        pos=pin((0.0, 0.0)),
    )

    # The loop. Colors alternate so neighbouring hops stay separable.
    brief >> step(BLUE, "brief") >> draft
    draft >> step(PURPLE, "preview") >> publish
    publish >> step(BLUE, "assign") >> play
    play >> step(ORANGE, "answers") >> evidence
    evidence >> step(TEAL, "synthesis") >> insight
    insight >> step(GREEN, "next intent") >> brief

    # M.I.A is the through-line: one agent per stage it owns, 120 apart on the
    # ring so the spokes read as a star rather than stray diagonals.
    mia >> spoke("Mission Agent") >> draft
    mia >> spoke("Tutor Agent") >> play
    mia >> spoke("Learning Agent") >> insight


def frame_box(scale: int) -> tuple[int, int, int, int]:
    """Slide area reserved for the graph (left, top, right, bottom), in canvas px."""
    return (
        FRAME_MARGIN_X * scale,
        HEADER_HEIGHT * scale,
        (LOGICAL_SIZE[0] - FRAME_MARGIN_X) * scale,
        (LOGICAL_SIZE[1] - FRAME_MARGIN_BOTTOM) * scale,
    )


def write_slide_png() -> None:
    """Fit the native Graphviz render under the header on an exact 4K 16:9 canvas."""
    raw_png = RAW_OUTPUT.with_suffix(".png")
    left, top, right, bottom = frame_box(PNG_SCALE)
    frame = (right - left, bottom - top)
    with Image.open(raw_png) as source:
        fitted = ImageOps.contain(
            source.convert("RGB"), frame, method=Image.Resampling.LANCZOS
        )

    canvas = Image.new("RGB", CANVAS_SIZE, "white")
    canvas.paste(
        fitted,
        (left + (frame[0] - fitted.width) // 2, top + (frame[1] - fitted.height) // 2),
    )
    draw = ImageDraw.Draw(canvas)
    draw.text((42 * PNG_SCALE, 22 * PNG_SCALE), TITLE, fill=INK, font=FONT_TITLE)
    draw.text((44 * PNG_SCALE, 66 * PNG_SCALE), SUBTITLE, fill=MUTED, font=FONT_SUBTITLE)
    canvas.save(OUTPUT, quality=95)
    raw_png.unlink()


def svg_number(value: str) -> float:
    """Parse a Graphviz SVG dimension such as ``1208pt``."""
    return float(
        "".join(character for character in value if character.isdigit() or character in ".-")
    )


def write_portable_svg() -> None:
    """Wrap the Graphviz SVG in 16:9 under the header and inline every icon."""
    raw_svg = RAW_OUTPUT.with_suffix(".svg")
    raw_root = ET.parse(raw_svg).getroot()
    svg_namespace = "http://www.w3.org/2000/svg"
    xlink_namespace = "http://www.w3.org/1999/xlink"
    ET.register_namespace("", svg_namespace)
    ET.register_namespace("xlink", xlink_namespace)

    for image in raw_root.iter(f"{{{svg_namespace}}}image"):
        href_key = f"{{{xlink_namespace}}}href"
        href = image.get(href_key) or image.get("href")
        if not href or href.startswith("data:"):
            continue
        icon_path = Path(href)
        if not icon_path.exists():
            continue
        mime_type = mimetypes.guess_type(icon_path.name)[0] or "image/png"
        encoded = base64.b64encode(icon_path.read_bytes()).decode("ascii")
        image.set(href_key, f"data:{mime_type};base64,{encoded}")
        image.attrib.pop("href", None)

    view_box = raw_root.get("viewBox")
    if view_box:
        _, _, raw_width, raw_height = map(float, view_box.split())
    else:
        raw_width = svg_number(raw_root.get("width", "1920"))
        raw_height = svg_number(raw_root.get("height", "1080"))

    left, top, right, bottom = frame_box(1)
    frame_width, frame_height = right - left, bottom - top
    scale = min(frame_width / raw_width, frame_height / raw_height)
    fitted_width = raw_width * scale
    fitted_height = raw_height * scale

    raw_root.set("x", f"{left + (frame_width - fitted_width) / 2:.2f}")
    raw_root.set("y", f"{top + (frame_height - fitted_height) / 2:.2f}")
    raw_root.set("width", f"{fitted_width:.2f}")
    raw_root.set("height", f"{fitted_height:.2f}")
    raw_root.set("preserveAspectRatio", "xMidYMid meet")

    outer = ET.Element(
        f"{{{svg_namespace}}}svg",
        {
            "width": str(LOGICAL_SIZE[0]),
            "height": str(LOGICAL_SIZE[1]),
            "viewBox": f"0 0 {LOGICAL_SIZE[0]} {LOGICAL_SIZE[1]}",
            "role": "img",
            "aria-label": "ClassGame content lifecycle",
        },
    )
    ET.SubElement(
        outer,
        f"{{{svg_namespace}}}rect",
        {"width": "100%", "height": "100%", "fill": "white"},
    )
    outer.append(raw_root)

    title = ET.SubElement(
        outer,
        f"{{{svg_namespace}}}text",
        {
            "x": "42",
            "y": "55",
            "fill": INK,
            "font-family": SANS,
            "font-size": "36",
            "font-weight": "700",
        },
    )
    title.text = TITLE
    subtitle = ET.SubElement(
        outer,
        f"{{{svg_namespace}}}text",
        {"x": "44", "y": "82", "fill": MUTED, "font-family": SANS, "font-size": "17"},
    )
    subtitle.text = SUBTITLE

    ET.ElementTree(outer).write(SVG_OUTPUT, encoding="utf-8", xml_declaration=True)
    raw_svg.unlink()


write_slide_png()
write_portable_svg()
print(f"Wrote {OUTPUT} ({CANVAS_SIZE[0]}x{CANVAS_SIZE[1]})")
print(f"Wrote {SVG_OUTPUT} (vector {LOGICAL_SIZE[0]}x{LOGICAL_SIZE[1]} viewBox)")
