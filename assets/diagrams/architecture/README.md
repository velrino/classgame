# ClassGame architecture diagram

The current diagram shows the modular code and optional services. It does not claim a live Cloud Run deployment. The old Product tour screenshots stay unchanged.

## Files

- `architecture_graphviz.py` — source for the current icon-led diagram.
- `architecture.png` — 3840×2160 PNG used by the root README and dashboard.

## Regenerate

From the repository root:

```sh
uv run assets/diagrams/architecture/architecture_graphviz.py
```

Graphviz must be installed. The script writes only PNG and syncs `web/public/assets/diagrams/architecture/architecture.png`. Keep the 16:9 frame and review the render after changes. The old manual renderer and duplicate images are removed so they cannot overwrite the current diagram.

## Current boundaries

- Browser modules call the Go API directly with Firebase ID tokens. The student game keeps pending evidence in IndexedDB. Selected Firestore projections provide scoped realtime reads; full history uses the paged API.
- One Go module owns classrooms, missions, tutoring, learning, insights, and attachments. Workflows use ports and provider adapters. Scalar exposes the OpenAPI reference.
- Interactive Mia tutoring is synchronous. All model workflows share Redis admission through the provider adapter. Vertex AI or the Gemini API selects the configured model.
- Optional Asynq workers use a dedicated Redis broker for copies, finalization, inactivity checks, and recovery. Delivery is at least once. Durable facts, replay, and claims stay in Firestore.
- Queue Redis is separate from Redis used for quotas and shared AI admission. Local jobs default off; finalization has an inline fallback.
- Firebase Auth verifies identity. Cloud Storage worksheets are private and downloads go through the authorized API. Workers copy validated temporary Uploadcare inputs. PostHog is optional and receives content-free events.

See [system architecture](../../../docs/architecture/SYSTEM.md) and [queue operations](../../../docs/operations/QUEUES.md) for details.
