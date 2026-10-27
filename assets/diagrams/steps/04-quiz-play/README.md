# 04 — Answer a quiz question with Mia

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

The student saves an answer locally before sending it to the API. Mia uses the **pinned question and bounded learning context** to give feedback. The API saves authoritative evidence before replying. The browser then flushes pending evidence and saves the next checkpoint. Action 14 returns accepted progress for resume.

## Before the first turn

| API call | Purpose |
| --- | --- |
| `POST /users/student-mission` | Select a published mission and pin `attempt_id` plus `published_revision` in the user session. This does not create the learning attempt document. |
| `GET /users/student-learning` | Load any existing checkpoint for that scope. |
| `POST /users/student-learning` | Save the initial checkpoint and create `student_mission_attempts/{attempt_id}` before tutoring starts. Later calls merge progress. |

Each tutor request uses `POST /users/student-tutor-turn` with a stable `event_id`, question, response, mission, attempt, and revision scope. Feedback returns as **synchronous JSON**.

## Evidence and retry rules

- The API derives the question, expected answer, and outcome from the pinned publication. Browser progress is presentation state; server-recorded accepted answers control completion.
- A completed event with matching content replays its saved result. Changed content conflicts. An active event returns `503 tutor_event_pending` for retry.
- New generated turns reserve an owner token with a **2-minute lease**. The final transaction rechecks scope, fingerprint, and ownership. The LLM call stays outside the transaction. Feedback-only observations need no LLM call.
- Missing knowledge, unavailable models, provider failure, or unsafe output can use deterministic fallback feedback. No general exactly-once LLM guarantee is made.
- Student replies omit teacher observations, internal usage, memory, and provider details. Playable question payloads include expected answers and explanations; answer keys are not hidden from the browser.

The DB stores the checkpoint and accepted answers in `student_mission_attempts/{attempt_id}`. Event reservations and replies live in its `tutor_events/{hashed_event_id}` subcollection. The pinned mission comes from `missions/{mission_id}/published_revisions/{revision}`.

## Bounded context and local recovery

Mia receives at most **5 knowledge chunks, 12 recent turns, and 24 recent events**. Student and class adaptation is reduced to allowed published skill IDs and explanation style. An attempt allows 36 generated admissions, including uncertain failed calls. Stored recent turns cap at 80; accepted answers remain separate. Rate checks and shared provider budgets also apply.

The browser saves pending evidence in IndexedDB before advancing. Limits are **64 records per scope, 256 total, 1 MiB total, and 16 KiB per record**. Storage failure or a full queue blocks new responses. Automatic retries pause after five failures; unsent work remains on the device. Checkpoints wait for pending tutor evidence to flush. Scope conflicts require mission reselection.

Display, tutor grounding, accepted-answer migration, and completion use the same pinned playable-question list, including short and legacy publications. Post-quiz finalization and learning-memory updates belong to the next diagram.

The `create participant` statements only place existing services at first use. See the [API contract](../../../../docs/reference/API.md) for route guidance.

Sources: [browser outbox](../../../../web/src/modules/learning/adapters/tutor-outbox.ts), [evidence sync](../../../../web/src/modules/learning/adapters/hooks/use-evidence-sync.ts), [tutoring](../../../../api/internal/modules/students/tutoring.go), [event reservation](../../../../api/internal/modules/students/firestore_tutor_reservation.go), [context limits](../../../../api/internal/modules/students/tutor_context.go), and [checkpoint service](../../../../api/internal/modules/students/learning.go).

## Regenerate

From the repository root:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/04-quiz-play/SEQUENCE.txt \
  -o assets/diagrams/steps/04-quiz-play/diagram.png \
  -b white -w 2200 -s 1.5
```

Use a white preview canvas and PNG export background.
