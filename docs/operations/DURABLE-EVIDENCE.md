# Durable learning evidence

The recent tutor window is context, not the full record. Each new accepted event has a private document under `student_mission_attempts/{attempt_id}/tutor_events/{turn_id}`. It stores the canonical request fingerprint and accepted response. Reusing an event ID with changed evidence returns `409 tutor_event_conflict`; an exact replay can return its saved response after it leaves the recent window.

Accepted correct-answer facts, event sequence and generation-admission count live on the attempt. Assessment checks facts against the pinned questions. Browser scores and old unverified snapshot progress do not award credit. Feedback cannot reset the count by pushing answer turns out of the 80-turn context window.

## Provider work

Before a new paid tutor call, a Firestore transaction checks attempt ownership/scope, the event fingerprint, the admission count and a two-minute event lease. One owner can commit the response. Other callers receive `503 tutor_event_pending` with `Retry-After: 5`, or the completed replay. Exact replays and feedback skip the paid Redis admission gate; the route still has bounded abuse protection.

The count measures durable admissions. A failed or crashed call may still have charged the provider, so its slot is not refunded. An expired lease permits a conservative retry with another admission. This reduces concurrent duplicate calls across API instances; it does not guarantee exactly-once provider billing.

## Finalization and recovery

A finalizer claims a five-minute lease. Its result and failure-state writes must match the owner token, attempt, student, classroom/tenant, pinned revision, status, snapshot revision and evidence revision. A resumed or changed attempt rejects stale results. An already finalized trace is preserved. Historical attempts may finish without changing the student's memory in another classroom.

A completion retry schedules pending work again using the same evidence-scoped job ID. Recovery scans use document-ID cursors, a narrow field projection and at most 100 documents per page. A page can contain no candidates and still return a cursor.

Read one page without writes, enqueueing or model calls:

```bash
cd api
go run ./cmd/learning-recovery --limit 25
# Pass the returned next_cursor to --after for the next page.
```

Worker roles scan one page of 50 documents per minute, with a 30-second deadline. A cursor advances even when a page has no candidates; the next scan starts again after the last page. Replicas may scan the same records, so stable job IDs and domain claims remain required. `RecoverPendingFinalizations` retries stable IDs through the configured queue. The default listing command does not start a worker. `RedrivePendingFinalizations` is an explicit operator action for exhausted/archived deliveries: it advances the dispatch generation atomically, skips active leases and then enqueues. Normal sweeps do not change dispatch generations.

## Legacy limits

Closed pre-upgrade completed records are historical and are not automatically regraded or backfilled. Existing server-recorded turns can seed only facts that still match pinned content. If an old 80-turn window may have lost earlier history, the known generation count is a lower bound and new paid admissions fail closed. Lost pre-upgrade answers, charges and replay responses cannot be invented. A teacher may need a new mission revision or an explicitly reviewed migration.

New event documents stay private for the attempt's retention lifetime. A retention change must account for replay guarantees and data deletion requirements.

This code change does not run recovery against shared data or deploy database rules. See the release validation record for emulator and offline checks; local builds alone do not prove a hosted workflow.

After reviewing the same page, explicit actions are available with the dedicated broker configured:

```sh
# Enqueue pending work with existing IDs; no model runs in this command.
go run ./cmd/learning-recovery --limit 25 --enqueue
# Only after reviewing archived/exhausted work; changes dispatch generations.
go run ./cmd/learning-recovery --limit 25 --redrive
```

Both actions require the same project/cursor settings as the reviewed listing. They can affect every eligible record in that bounded page. Stop or coordinate concurrent operator runs. Workers may process the newly queued jobs immediately; these commands are operational writes, never part of the offline evaluation suite.
