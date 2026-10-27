# 01 — Upload material and prepare a mission draft

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

The teacher uploads **optional material** to Uploadcare. The API **checks access**, downloads supported files, requests short LLM summaries, generates the mission, and **saves the draft in Firestore**. **Action 13** returns the preview for teacher review.

All participants use normal boxes, white labels, and a blue-to-purple palette on a white background. Boxes appear at first use. The `create participant` commands control this static layout; the services already exist. **Without attachments**, the API skips file summaries and still generates a draft.

## Regenerate

Run from the repository root with Node.js and Mermaid CLI available:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/01-mission-draft/SEQUENCE.txt \
  -o assets/diagrams/steps/01-mission-draft/diagram.png \
  -b white -w 2200 -s 1.5
```

You can also paste the full source into [Mermaid Live Editor](https://mermaid.live/). Use its light appearance and a white PNG export background.
