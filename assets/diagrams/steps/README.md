# Mission flow diagrams

Each numbered step folder owns its `README.md`, `SEQUENCE.txt`, and one `diagram.png`.

Select a diagram to open the full PNG. Add a row when a new step is ready.

| Step | Flow | Diagram |
|:---:|---|:---:|
| 01 | **Upload material and prepare a mission draft**<br>[Guide](01-mission-draft/README.md) · [Mermaid source](01-mission-draft/SEQUENCE.txt) | <a href="01-mission-draft/diagram.png"><img src="01-mission-draft/diagram.png" alt="Sequence diagram for uploading material and preparing a mission draft" width="480" /></a> |

This step ends at action 13, when the API returns the saved draft for teacher review. Publication, private file-copy workers, quiz play, and learning memory will have separate diagrams.

## Style

Use a white background, normal participant boxes, white participant labels, and Arial. Introduce existing services visually at first use. Keep one short flow per diagram.

## Palette for future steps

Use this approved palette in order:

```text
#04c7ff
#5fcefd
#86d5fb
#a5dcf8
#c0e3f6
#d9eaf3
#f1f1f1
#ded0e4
#ccb1d6
#b892c9
#a573bc
#9054af
#7b34a1
```

The [system architecture](../../../docs/architecture/SYSTEM.md) describes the full application.
