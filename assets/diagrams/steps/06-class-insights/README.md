# 06 — Refresh class memory and teacher insights

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

After [step 05](../05-learning-memory/README.md), a durable class-refresh intent identifies the evidence generation to summarize. The worker builds **bounded class memory** and saves it only while its owner, lease, tenant, and source generation still match. The summary change invalidates the open teacher dashboard. Action 15 returns the authorized overview for the visible student page.

## Refresh and memory bounds

The queued job contains the class ID, source generation, and dispatch generation. A transaction claims the pending generation with an owner token and a **5-minute lease**. The synthesis input contains at most **100 latest finalized student evidence records**. The repository checks each record against its source attempt and class tenant.

The model receives bounded names, IDs, strengths, and needs from that evidence. Output keeps at most **4 strength clusters and 4 need clusters**, with labels up to 180 characters and next adaptation up to **280 characters**. Known evidence labels and student IDs are checked. Missing evidence, unavailable models, or provider errors can use an observed deterministic fallback.

| Record | Role |
| --- | --- |
| `class_learning_students/{class_id}/students/{student_id}` | Latest finalized evidence for each student. |
| `class_learning_refreshes/{class_id}` | Source/completed generations, dispatch generation, owner token, and lease. |
| `class_learning_summaries/{class_id}` | Bounded synthesis, source generation, fingerprint, and evidence timestamp. |

The final transaction writes the summary and advances `completed_generation` together. A changed generation, superseded owner, expired lease, or changed class scope prevents that result from committing. Old/completed jobs do no work. Quota admission and processing failures leave current intent pending for retry or recovery. Recovery is independent from whether the original student attempt is already finalized. See [queue operations](../../../../docs/operations/QUEUES.md).

With jobs disabled, the same processor can run inline. Enqueue failure leaves the durable intent available to recovery. Evidence updates and removals advance its source generation, so matching timestamps or row counts cannot authorize an older result.

## What the teacher sees

An open dashboard listens to the single summary document. A change triggers a new **authenticated API read**; visible pages also refresh every **15 seconds** and when the tab becomes visible. Firestore rules restrict that subscription to the class owner. API reads check teacher identity, class ownership, and tenant evidence.

The normal route is `GET /dashboard/overview/page`, with a default page of **25 students** and an accepted limit of 1–100. It returns current-page patterns and latest-attempt summaries. The summary document is an **invalidation signal** here: the paged route does not directly apply the stored semantic synthesis.

The capped legacy route, `GET /dashboard/overview`, can apply that synthesis. The browser uses it only for the documented index-error fallback, within its 100-student/100-attempt caps. Background refresh failures preserve the last successful page.

This is class-level memory and a teacher-facing view. Separate teacher, school, and district AI-memory stores are outside the current scope. Historical evidence remains separate from the compact summary. The `create participant` commands only place existing participants at first use.

Sources: [refresh processor](../../../../api/internal/modules/students/class_refresh.go), [transaction fences](../../../../api/internal/modules/students/firestore_class_refresh.go), [bounded synthesis](../../../../api/internal/modules/generation/class_synthesis.go), [overview service](../../../../api/internal/modules/insights/service.go), [realtime invalidation](../../../../web/src/modules/insights/adapters/realtime.ts), [visible-page refresh](../../../../web/src/modules/insights/adapters/hooks/use-dashboard-snapshots.ts), and [API adapter](../../../../web/src/modules/insights/adapters/http.ts).

## Regenerate

From the repository root:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/06-class-insights/SEQUENCE.txt \
  -o assets/diagrams/steps/06-class-insights/diagram.png \
  -b white -w 2200 -s 1.5
```

Use a white preview canvas and PNG export background.
