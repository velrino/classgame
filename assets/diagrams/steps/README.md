# Mission flow diagrams

Each numbered step folder owns its `README.md`, `SEQUENCE.txt`, and one `diagram.png`.

Select a diagram to open the full PNG. Add a row when a new step is ready.

| Step | Flow | Diagram |
|:---:|---|:---:|
| 01 | **Upload material and prepare a mission draft**<br>[Guide](01-mission-draft/README.md) · [Mermaid source](01-mission-draft/SEQUENCE.txt) | <a href="01-mission-draft/diagram.png"><img src="01-mission-draft/diagram.png" alt="Sequence diagram for uploading material and preparing a mission draft" width="480" /></a> |
| 02 | **Review and publish the mission**<br>[Guide](02-mission-publication/README.md) · [Mermaid source](02-mission-publication/SEQUENCE.txt) | <a href="02-mission-publication/diagram.png"><img src="02-mission-publication/diagram.png" alt="Sequence diagram for reviewing and publishing a mission" width="480" /></a> |
| 03 | **Copy mission files to private storage**<br>[Guide](03-attachment-transfer/README.md) · [Mermaid source](03-attachment-transfer/SEQUENCE.txt) | <a href="03-attachment-transfer/diagram.png"><img src="03-attachment-transfer/diagram.png" alt="Sequence diagram for copying mission attachments to private storage" width="480" /></a> |
| 04 | **Answer a quiz question with Mia**<br>[Guide](04-quiz-play/README.md) · [Mermaid source](04-quiz-play/SEQUENCE.txt) | <a href="04-quiz-play/diagram.png"><img src="04-quiz-play/diagram.png" alt="Sequence diagram for a quiz turn, Mia feedback, and checkpoint saving" width="480" /></a> |

Step 01 ends with the saved preview for teacher review. Step 02 ends with the published revision. Step 03 records private file-copy outcomes. Step 04 saves quiz evidence and progress. Post-quiz finalization and learning memory will have separate diagrams.

## Style

Use a white background, normal participant boxes, white participant labels, and Arial. Introduce existing services visually at first use. Keep one short flow per diagram. Use bold for key actions, responses, and note headings; keep supporting text regular.

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
