# Data ownership

Firestore is the durable store. Records carry the configured organization, district, and school scope; classroom operations also enforce teacher ownership or student enrollment. The demo's default tenant is configuration, not a multi-tenant authorization model.

| Collection or path | Purpose |
| --- | --- |
| `tenants`, `districts`, `schools` | Tenant hierarchy |
| `teachers/{uid}` | Teacher identity, profile, roles, activity |
| `teacher_classes/{class_id}` | Ownership, class code, subject, active mission, status |
| `missions/{mission_id}` | Draft/published snapshots, revisions, publication lifecycle |
| `missions/{mission_id}/published_revisions/{revision}` | Immutable published content and attachment-transfer state |
| `mission_codes/{mission_code}` | Reserved mission-code lookup where used |
| `student_class_missions/{class_id}/missions/{mission_id}` | Student-safe published mission catalog |
| `users/{uid}` | Student session, presence, durable learning memory |
| `student_mission_attempts/{attempt_id}` | Pinned revision, checkpoints, events, tutor turns, learning trace |
| `student_mission_completions/{student_id}/missions/{attempt_id}` | Minimal student-readable completion projection |
| `class_learning_students/{class_id}` | Rebuild generation, cursor, and completion state |
| `class_learning_students/{class_id}/students/{student_id}` | Private latest finalized evidence; source-version checked |
| `class_learning_summaries/{class_id}` | Class-wide evidence synthesis |

Legacy class-scoped mission reads may coexist during migration. New writes and migrations must state which representation is authoritative before retiring compatibility reads.

## Publication

A mission retains separate draft and published content. Approval checks the expected draft revision and transactionally creates its immutable revision, updates the current mission, activates it for the classroom, and refreshes the student catalog. Existing attempts keep their pinned content. Soft deletion removes discoverability while preserving history needed by attempts and background jobs.

## Learning

Attempts identify the student, class, mission, and published revision. Resume state includes a revisioned checkpoint. Observable events and tutor turns support evidence traces; student memory records supported strengths, needs, explanation preference, and the next adaptation.

Treat browser snapshots as untrusted input. Derive authoritative progress from accepted events against the pinned mission rather than trusting client completion flags. Finalized evidence and duplicate event/task identities need explicit replay behavior.

## Browser access

[Firestore rules](../../firestore.rules) deny browser writes. Students consume explicit projections and student-safe API responses; teachers read authorized class evidence. Class documents, private observations, and provider storage documents must not become general frontend contracts.

Introduce new projections additively: write/backfill, verify, switch consumers, then retire old access. Include projection version/freshness and a rebuild procedure for derived insights. Do not change deployed rules until the corresponding consumers and backfill are ready.

Teacher overview and history use [bounded pages and rebuildable evidence](../operations/QUERY-SCALE.md). Page counts are not whole-class totals. New projection reads switch only after a complete class rebuild.
