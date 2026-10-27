# 05 — Finish the quiz and refresh student memory

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

After [step 04](../04-quiz-play/README.md), the browser flushes pending tutor evidence and requests completion. The API confirms completion from **server-recorded accepted answers**. A finalizer then builds a private learning trace and replaces eligible student memory with a **compact summary**. Action 15 completes the queued delivery.

## Completion and readiness

`POST /users/student-learning` sends `status: "completed"`, the snapshot, and mission/attempt/revision scope. The server can keep the attempt in progress when accepted evidence is insufficient. The reward waits for an acknowledged `completed` response and an empty, unblocked outbox. It does not wait for AI-memory readiness.

The diagram shows the queued path. With jobs disabled or enqueue failure, the API runs the **same finalizer inline**. The student response contains resume data, not teacher observations, finalization internals, or memory. Teacher insights refresh separately; the student browser does not poll AI readiness.

## Stored records

| Record | Finalization behavior |
| --- | --- |
| `student_mission_attempts/{attempt_id}` | Record completion, pending/processing/ready trace state, owner, and source versions. |
| `student_mission_completions/{student_id}/missions/{attempt_id}` | Store the authoritative completion marker. |
| `class_learning_students/{class_id}/students/{student_id}` | Update the teacher-facing evidence projection when class scope still matches. |
| `users/{student_id}.student.learning_memory` | Replace memory only for the current class/tenant and evidence no older than the saved memory. |

The final transaction saves the private trace. It also writes class evidence and student memory when their scope checks pass. The class summary at `class_learning_summaries/{class_id}` refreshes afterward as a separate operation.

## Bounded memory and recovery

Student memory keeps at most **4 strengths and 4 needs**, up to 180 characters each, plus next adaptation up to **280 characters**. Trace input uses at most 160 recent events, 12 progress entries, 40 notes, and 40 decisions. Previous-memory input is also capped. A later tutor turn receives filtered skill IDs and explanation style. History remains separate; compact memory does not delete historical evidence. See [retention and recovery](../../../../docs/operations/RETENTION-RECOVERY.md).

The finalizer claims a random owner token and a **5-minute lease**. Its commit compares student/class/tenant scope, publication, status, snapshot/evidence versions, dispatch generation, and owner token. Lease expiry permits a new claim; the final write uses that captured owner/version fence. Finalized or superseded jobs do no new finalization work.

Worker recovery scans bounded pages of 50 rows, once per minute, excluding active claims and finalized attempts. Archived or exhausted job IDs need reviewed redrive. Missing models/events, quota limits, or provider failure can use an observed deterministic trace. Other returned processing errors remain retryable. See [queue recovery](../../../../docs/operations/QUEUES.md).

## Current memory scopes

| Scope | Current support |
| --- | --- |
| Student | Compact memory replaced by eligible finalization. |
| Class | Finalized student evidence plus a separate class summary; its flow is the next step. |
| Teacher | Private trace and class-summary views. Separate teacher-owned memory is a [design follow-up #98](https://github.com/velrino/classgame/issues/98). |
| School | Tenant boundary; separate aggregate memory is a [design follow-up #99](https://github.com/velrino/classgame/issues/99). |
| District | Tenant boundary; separate aggregate memory is a [design follow-up #100](https://github.com/velrino/classgame/issues/100). |

Class refresh can be missed after successful student finalization: [issue #96](https://github.com/velrino/classgame/issues/96). Concurrent summaries also need a stronger evidence-set fence: [issue #97](https://github.com/velrino/classgame/issues/97). These runtime follow-ups are separate from this diagram.

The `create participant` statements only place existing services at first use. Sources: [completion service](../../../../api/internal/modules/students/learning.go), [finalizer](../../../../api/internal/modules/students/learning_jobs.go), [claim and failure writes](../../../../api/internal/modules/students/firestore_finalization.go), [final commit](../../../../api/internal/modules/students/firestore_attempts.go), [fence](../../../../api/internal/modules/students/finalization.go), and [browser completion](../../../../web/src/modules/learning/adapters/hooks/use-learning-recovery.ts).

## Regenerate

From the repository root:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/05-learning-memory/SEQUENCE.txt \
  -o assets/diagrams/steps/05-learning-memory/diagram.png \
  -b white -w 2200 -s 1.5
```

Use a white preview canvas and PNG export background.
