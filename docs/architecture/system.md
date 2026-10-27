# System architecture

ClassGame is a modular monolith: a Next.js teacher dashboard and student game call one Go API. Firebase authenticates teachers and anonymous students. Firestore stores durable state; selected browser projections update through realtime subscriptions.

The submitted deployment used Cloud Run, Vertex AI, Cloud Storage, Cloud Tasks, Redis, Secret Manager, and Cloud Build. Hosting was shut down; see [hosting history](../operations/hosting-history.md).

![Original production architecture](../../assets/diagrams/architecture/architecture-graphviz.svg)

## Learning loop

1. A teacher creates a class and asks Mia for a mission, optionally attaching a worksheet.
2. The API prepares attachment context and generates a draft with questions, knowledge chunks, and a preview flow.
3. Approval publishes an immutable revision and its student catalog entry atomically.
4. A student joins by class code, selects a mission, and plays its pinned revision.
5. Tutor turns combine the current question, retrieved mission chunks, observable history, student memory, and class context.
6. Checkpoints restore the exact session. Completion or abandonment produces a learning trace and class insights for the teacher.

## Ownership

| Capability | Owns |
| --- | --- |
| Identity | Actor identity, accepted providers, authentication boundary |
| Classrooms | Ownership, enrollment, join codes, presence |
| Missions | Drafts, revisions, publication, student catalog |
| Learning | Attempts, progress, checkpoints, finalization |
| Tutoring | Turn orchestration, retrieval, deterministic policies |
| Insights | Evidence traces, student memory, class summaries, bounded teacher queries |
| Attachments | Ingestion, download validation, storage copies, authorized delivery |
| Game | Scene, controls, avatars, audio, world bridge |

Provider connections and telemetry are infrastructure. Application composition wires implementations; business modules consume narrow contracts. See [module anatomy](modules.md).

## Invariants

- Publishing the revision, active classroom mission, and student catalog is one transaction. A stale draft revision returns a conflict.
- An attempt remains pinned to its published revision after later edits or publication.
- Firebase identity verification does not replace classroom ownership or enrollment checks.
- Resume checkpoints and authoritative learning evidence have separate trust levels.
- Student responses and browser projections exclude teacher-only observations and administrative fields.
- Task retries use stable identities and preserve finalization state. External calls stay outside retryable Firestore transaction callbacks.
- Subscriptions, requests, and Three.js resources are released when scope, identity, or route changes.

## Agent workflows

Mission generation produces stages, multiple-choice questions, grounded knowledge chunks, guardrail results, and a preview graph. Tutor generation returns student-facing teaching content plus separately persisted teacher observations and retrieval references. Learning analysis summarizes observable interactions; class synthesis groups supported strengths and needs.

Each workflow has explicit budgets and deterministic fallback behavior. Retrieval currently uses lexical overlap over mission chunks. Prompts and structured output do not establish authorization or guarantee factual grounding; validate generated references and keep private context out of student projections.

Measure request latency, Firestore reads, task retries, AI fallback rates, and projection lag using identifiers rather than student or worksheet content.
