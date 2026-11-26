"""Write portable SVG and Excalidraw documents from the AI map's drawing calls."""

import base64
import hashlib
import json
import xml.etree.ElementTree as ET


class AIScene:
    def __init__(self, width, height):
        self.svg = ET.Element("svg", xmlns="http://www.w3.org/2000/svg",
                              width=str(width), height=str(height), viewBox=f"0 0 {width} {height}")
        ET.SubElement(self.svg, "title").text = "ClassGame AI system design"
        ET.SubElement(self.svg, "rect", width="100%", height="100%", fill="white")
        self.elements = []
        self.files = {}
        self.group = None

    def element(self, kind, x, y, w, h, color, fill="transparent", **props):
        number = len(self.elements) + 1
        self.elements.append(dict(
            id=f"ai-{number}", type=kind, x=x, y=y, width=w, height=h, angle=0,
            strokeColor=color, backgroundColor=fill, fillStyle="solid", strokeWidth=1,
            strokeStyle="solid", roughness=0, opacity=100, groupIds=[self.group] if self.group else [],
            frameId=None, roundness=None, seed=number, version=1, versionNonce=number,
            isDeleted=False, boundElements=None, updated=1, link=None, locked=False,
        ) | props)

    def rect(self, x, y, w, h, fill, stroke, radius):
        ET.SubElement(self.svg, "rect", x=str(x), y=str(y), width=str(w), height=str(h),
                      rx=str(radius), fill=fill, stroke=stroke)
        self.element("rectangle", x, y, w, h, stroke, fill, roundness={"type": 3})

    def text(self, x, y, text, size, width, color, bold):
        ET.SubElement(self.svg, "text", x=str(x), y=str(y+size), fill=color, attrib={
            "font-family": "Arial, Helvetica, sans-serif", "font-size": str(size),
            "font-weight": "bold" if bold else "normal"}).text = text
        self.element("text", x, y, width, size*1.25, color, fontFamily=2, fontSize=size,
                     text=text, originalText=text, textAlign="left", verticalAlign="top",
                     containerId=None, autoResize=True, lineHeight=1.25)

    def icon(self, path, x, y, size):
        data = path.read_bytes()
        key = hashlib.sha256(data).hexdigest()
        url = "data:image/svg+xml;base64," + base64.b64encode(data).decode("ascii")
        self.files[key] = dict(id=key, mimeType="image/svg+xml", dataURL=url, created=1)
        ET.SubElement(self.svg, "image", x=str(x), y=str(y), width=str(size), height=str(size),
                      attrib={"{http://www.w3.org/1999/xlink}href": url})
        self.element("image", x, y, size, size, "transparent", fileId=key,
                     status="saved", scale=[1, 1], crop=None)

    def arrow(self, x, y, color):
        ET.SubElement(self.svg, "path", d=f"M{x} {y} v22", stroke=color,
                      attrib={"stroke-width": "2"})
        ET.SubElement(self.svg, "path", d=f"M{x-4.5} {y+17} h9 l-4.5 7 z", fill=color)
        self.element("arrow", x, y, 0, 24, color, points=[[0, 0], [0, 24]],
                     startBinding=None, endBinding=None, startArrowhead=None, endArrowhead="arrow",
                     lastCommittedPoint=None, elbowed=False)

    def save(self, folder):
        ET.indent(self.svg)
        ET.ElementTree(self.svg).write(folder / "AI-SYSTEM.svg", encoding="utf-8", xml_declaration=True)
        scene = dict(type="excalidraw", version=2, source="https://excalidraw.com",
                     elements=self.elements, files=self.files,
                     appState=dict(viewBackgroundColor="#ffffff", gridSize=None))
        (folder / "AI-SYSTEM.excalidraw").write_text(json.dumps(scene, indent=2) + "\n")
