# 01 — Upload material and prepare a mission draft

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

## Request and response

| Request | Response | Why |
| --- | --- | --- |
| Teacher → Uploadcare: multipart file upload. | JSON: UUID, CDN URL, name, MIME, size. | Upload once; pass file references. |
| Teacher → API: JSON goal + attachment metadata; `POST /teacher-classes/{class}/missions/preview/stream`. | SSE: JSON progress events + saved mission preview. | Show progress while the LLM works; no Base64 in this request. |
| API → Gemini through LangChainGo | A mission draft with three questions. | A single readable PDF goes directly into this call with the teacher goal. |
| API → material workflow for other supported files | Short summaries for the mission workflow. | Each summary gives the generator bounded context. |

With no files, the mission workflow uses the teacher request. If a file cannot be read, it uses available metadata and context. The API validates the result, saves the draft in Firestore and sends the final preview event.

## Regenerate

Run from the repository root with Node.js and Mermaid CLI available:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/01-mission-draft/SEQUENCE.txt \
  -o assets/diagrams/steps/01-mission-draft/diagram.png \
  -b white -w 2200 -s 1.5
```

You can also paste the full source into [Mermaid Live Editor](https://mermaid.live/). Use its light appearance and a white PNG export background.
