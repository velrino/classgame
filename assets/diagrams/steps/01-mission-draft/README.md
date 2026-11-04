# 01 — Upload material and prepare a mission draft

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

## Request and response

| Request | Response | Why |
| --- | --- | --- |
| Teacher → Uploadcare: multipart file upload. | JSON: UUID, CDN URL, name, MIME, size. | Upload once; pass file references. |
| Teacher → API: JSON goal + attachment metadata; `POST /teacher-classes/{class}/missions/preview/stream`. | SSE: JSON progress events + saved mission preview. | Show progress while the LLM works; no Base64 in this request. |
| API → LLM: prompt + file in `inlineData` (MIME + Base64 bytes). | JSON: short attachment summary. | Carry original file bytes inside JSON. |
| API → LLM: goal + attachment summaries in the prompt. | JSON: questions, answers, explanations, stages. | Generate the draft without sending the full file again. |

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
