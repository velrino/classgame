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

    uv run architecture.py

Outputs beside this file:

- architecture.png — 4K PNG image, also copied to the web app.
"""

from pathlib import Path

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
OUTPUT = ROOT / "architecture.png"
WEB_PNG_OUTPUT = ROOT.parents[2] / "web/public/assets/diagrams/architecture/architecture.png"

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
    outformat="png",
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



write_slide_png()
WEB_PNG_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
WEB_PNG_OUTPUT.write_bytes(OUTPUT.read_bytes())
print(f"Wrote {OUTPUT} ({CANVAS_SIZE[0]}x{CANVAS_SIZE[1]})")
print(f"Synced {WEB_PNG_OUTPUT}")
