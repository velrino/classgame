# /// script
# requires-python = ">=3.11"
# dependencies = ["diagrams>=0.25,<0.26", "pillow>=11,<13"]
# ///
"""Render an icon-led ClassGame architecture with mingrammer/diagrams.

Graphviz owns the topology; this script only shapes it: nodes are compact
icon cards (icon on top, bold title, muted subtitle), clusters are quiet
rounded panels, and the finished graph is fitted below a slide header on an
exact 16:9 canvas. Nothing is stretched, so type stays readable.

Run from any directory:

    uv run architecture_graphviz.py

Outputs beside this file:

- architecture-graphviz.png — 4K raster image.
- architecture-graphviz.svg — self-contained vector image.
"""

import base64
import mimetypes
from pathlib import Path
import xml.etree.ElementTree as ET

from diagrams import Cluster, Diagram, Edge, Node
from diagrams.gcp.database import Firestore
from diagrams.gcp.ml import AIPlatform, VertexAI
from diagrams.gcp.operations import Monitoring
from diagrams.gcp.security import Iam
from diagrams.gcp.storage import GCS
from diagrams.onprem.inmemory import Redis
from diagrams.programming.flowchart import Decision, Document
from diagrams.programming.framework import React
from diagrams.programming.language import Go
from PIL import Image, ImageDraw, ImageFont, ImageOps


ROOT = Path(__file__).resolve().parent
RAW_OUTPUT = ROOT / "architecture-graphviz-raw"
OUTPUT = ROOT / "architecture-graphviz.png"
SVG_OUTPUT = ROOT / "architecture-graphviz.svg"
WEB_SVG_OUTPUT = ROOT.parents[2] / "web/public/assets/diagrams/architecture/architecture-graphviz.svg"

LOGICAL_SIZE = (1920, 1080)
PNG_SCALE = 2
CANVAS_SIZE = tuple(dimension * PNG_SCALE for dimension in LOGICAL_SIZE)
# Slide header band (logical px). The graph is fitted into the space below it.
HEADER_HEIGHT = 112
FRAME_MARGIN_X = 48
FRAME_MARGIN_BOTTOM = 64

TITLE = "ClassGame · Modular Architecture"
SUBTITLE = "One Next.js app · One Go module · Durable evidence · Private worksheets"
FOOTER = "Scoped realtime only · History through the paged API · Local jobs default off · No hosted deployment implied"

BLUE = "#3b63f3"
PURPLE = "#8b5cf6"
ORANGE = "#ed8b00"
TEAL = "#0ea5b7"
GREEN = "#16a368"
RED = "#dc382d"
INK = "#211833"
MUTED = "#645b73"
PANEL_FILL = "#f6f7fb"
PANEL_LINE = "#d8dce8"
SANS = "Helvetica Neue, Helvetica, Arial, sans-serif"


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    """Load the same title fonts used by the hand-laid-out architecture."""
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
FONT_FOOTER = font(13 * PNG_SCALE)

# No `ratio`/`size`: Graphviz lays the graph out at its natural density and we
# fit the result onto the slide afterwards, so spacing and type scale together.
graph_attr = {
    "bgcolor": "white",
    "dpi": "192",
    "pad": "0.3",
    "nodesep": "0.42",
    "ranksep": "1.25",
    "splines": "spline",
    "newrank": "true",
    "compound": "true",
    # Best effort only: dot keeps a node's out-edges in declaration order, but
    # heads inside different clusters may still be reordered.
    "ordering": "out",
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
    "penwidth": "1.6",
    "arrowsize": "0.75",
    "labeldistance": "1.2",
}
cluster_attr = {
    "style": "rounded,filled",
    "fillcolor": PANEL_FILL,
    "bgcolor": PANEL_FILL,
    "pencolor": PANEL_LINE,
    "penwidth": "1.2",
    "fontname": "Helvetica-Bold",
    "fontsize": "13",
    "fontcolor": MUTED,
    "labeljust": "r",
    "margin": "22",
}


def card(icon: type[Node], title: str, subtitle: str, **attrs) -> Node:
    """Icon on top, bold title, muted subtitle — one compact card per component."""
    label = (
        f'<<font point-size="16"><b>{title}</b></font>'
        f'<br/><font point-size="12" color="{MUTED}">{subtitle}</font>>'
    )
    return icon(
        label,
        width="1.45",
        height="2.0",
        fixedsize="true",
        imagescale="true",
        imagepos="tc",
        labelloc="b",
        **attrs,
    )


def panel(title: str) -> Cluster:
    return Cluster(title, graph_attr=cluster_attr)


def flow(color: str = BLUE, label: str = "", **attrs) -> Edge:
    return Edge(color=color, label=label, **attrs)


def pulse(color: str, label: str = "", **attrs) -> Edge:
    """Realtime / telemetry traffic: same weight, dashed."""
    return Edge(color=color, label=label, style="dashed", **attrs)


with Diagram(
    "",
    filename=str(RAW_OUTPUT),
    direction="LR",
    show=False,
    outformat=["png", "svg"],
    graph_attr=graph_attr,
    node_attr=node_attr,
    edge_attr=edge_attr,
):
    with panel("NEXT.JS APP · BROWSER MODULES"):
        student_game = card(React, "Student game", "Three.js · IndexedDB outbox")
        dashboard = card(React, "Teacher dashboard", "adapters · hooks · UI")

    # These workflows are packages in the same Go module, not microservices.
    with panel("GO MODULE · IN-PROCESS WORKFLOWS"):
        api = card(Go, "Go API", "auth · classes · Scalar")
        mission = card(AIPlatform, "Mission workflow", "draft · publish · attachments")
        tutor = card(AIPlatform, "Mia tutor", "synchronous · grounded context")
        learning = card(AIPlatform, "Learning + insights", "evidence · traces · summaries")
        policy = card(Decision, "AI policy + ports", "validation · bounded fallback")

        api >> flow(label="preview SSE · publish") >> mission
        api >> flow(label="answer · feedback") >> tutor
        api >> flow(label="checkpoint · paged history") >> learning
        mission >> flow(PURPLE) >> policy
        tutor >> flow(PURPLE) >> policy
        learning >> flow(PURPLE) >> policy

    with panel("MODEL PROVIDER · CONFIGURED"):
        model = card(VertexAI, "Gemini", "Vertex AI or Gemini API")

    with panel("BACKGROUND JOBS · OPTIONAL"):
        broker = card(Redis, "Job broker", "dedicated Redis · Asynq")
        workers = card(Go, "Go workers", "copy · finalize · inactivity")
        broker >> flow(ORANGE, label="at-least-once delivery") >> workers

    with panel("SHARED CONTROLS"):
        limits = card(Redis, "Shared limits", "quotas · AI admission")
        posthog = card(Monitoring, "PostHog", "optional · content-free events")

    with panel("IDENTITY + DURABLE DATA") as data_panel:
        data_panel.dot.attr(rank="same")
        firebase_auth = card(Iam, "Firebase Auth", "teacher policy · anonymous")
        firestore = card(Firestore, "Firestore", "revisions · evidence · claims")
        storage = card(GCS, "Private worksheets", "Cloud Storage · API downloads")

    with panel("UPLOAD INTAKE"):
        uploadcare = card(Document, "Uploadcare", "temporary direct uploads")

    # Both browser experiences call the API directly; Next.js is not a proxy.
    student_game >> flow(label="HTTPS · Firebase ID token", weight="6") >> api
    dashboard >> flow(label="HTTPS · Firebase ID token", weight="6") >> api
    api >> flow(label="verify identity", minlen="2") >> firebase_auth
    api >> flow(label="state · safe projections", weight="5") >> firestore
    api >> flow(label="authorized downloads") >> storage

    # The queue owns delivery only. Durable facts and claims stay in Firestore.
    api >> flow(ORANGE, label="enqueue scoped jobs") >> broker
    api >> Edge(style="invis") >> uploadcare
    workers >> flow(ORANGE, label="leases · finalize") >> firestore
    workers >> flow(ORANGE, label="private copy") >> storage
    uploadcare >> flow(ORANGE, label="validated source") >> workers
    workers >> pulse(ORANGE, label="same Go services", constraint="false") >> learning

    # All model calls use the same provider port and shared Redis admission.
    policy >> flow(PURPLE, label="bounded prompt · JSON", weight="4") >> model
    policy >> flow(RED, label="reserve · release") >> limits
    workers >> flow(RED, label="shared admission", constraint="false") >> limits
    api >> pulse(TEAL, label="optional telemetry", minlen="2") >> posthog

    # Realtime subscriptions use scoped Firestore projections. Their long
    # browser-to-data arcs are summarized in the footer to keep labels legible.


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
    draw.text((44 * PNG_SCALE, 1058 * PNG_SCALE), FOOTER, fill=MUTED, font=FONT_FOOTER)
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
            "aria-label": "ClassGame current modular architecture",
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

    footer = ET.SubElement(
        outer,
        f"{{{svg_namespace}}}text",
        {"x": "44", "y": "1069", "fill": MUTED, "font-family": SANS, "font-size": "13"},
    )
    footer.text = FOOTER
    ET.ElementTree(outer).write(SVG_OUTPUT, encoding="utf-8", xml_declaration=True)
    WEB_SVG_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    WEB_SVG_OUTPUT.write_bytes(SVG_OUTPUT.read_bytes())
    raw_svg.unlink()


write_slide_png()
write_portable_svg()
print(f"Wrote {OUTPUT} ({CANVAS_SIZE[0]}x{CANVAS_SIZE[1]})")
print(f"Wrote {SVG_OUTPUT} (vector {LOGICAL_SIZE[0]}x{LOGICAL_SIZE[1]} viewBox)")
print(f"Synced {WEB_SVG_OUTPUT}")
