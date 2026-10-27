# Bounded classroom queries

The teacher app reads one student page and loads full history only when a teacher opens a student. It does not download all attempts through a realtime listener.

## Read contract

| Read | Bound and meaning |
| --- | --- |
| `/dashboard/overview/page` | Default 25, maximum 100 students; one latest attempt summary per student |
| Roster query | At most `limit + 1` rows; one extra row detects the next page |
| Latest attempt summaries | At most one row per visible student; no more than 8 queries run together |
| Student history | Default 5, maximum 10 full attempts; at most `limit + 1` rows |
| Old `/dashboard/overview` | Full response only within 100 student rows and 100 attempts; otherwise 409 `pagination_required` |
| Class AI context | At most 100 latest evidence students; normal reads validate at most twice that number of rows |

The overview response labels its counts and patterns `student_page`. It includes `read_at` and a next cursor. The UI shows that scope on every tab, provides Previous/Next controls, and refreshes the visible page every 15 seconds. History pages show their read time and have their own refresh button. Older attempts remain reachable. If new indexes are not ready, the web app can use the complete capped compatibility view for small classes, including their embedded full history. It never treats a truncated sample as that fallback. Large classes retain the clear index error until the approved rollout is ready. The compatibility route uses its existing class/document-ID indexes and sorts only a complete capped result; it does not need the new per-student time indexes.

Roster cursors use document IDs. History cursors use last activity and document ID, and are bound to the class, student, and route. Ties cannot skip duplicate timestamps. Each request is a fresh database read; active records can move as new work arrives. This is not a consistent export of the entire class.

## Latest finalized evidence

`class_learning_students/{class}/students/{student}` is a private, versioned projection with the attempt ID, tenant/class/student scope, evidence and snapshot versions, source time, and short strength/need labels. No raw worksheet, prompt, or conversation is copied.

A successful finalizer writes its projection in the same transaction as its fenced result. Projection reads validate the current source's tenant, class, student, pinned mission/revision, status, versions, timestamp, and finalized flag. A resumed source is excluded. A late worker or older rebuild row cannot replace newer valid evidence.

The parent `class_learning_students/{class}` records the rebuild generation, cursor, and completion flag. Reads switch to the projection only after all rebuild pages finish in order. A concurrent reset invalidates an old rebuild's progress. During migration, AI context uses a bounded window of at most `4 * limit` recent finalized source attempts, then selects at most `limit` distinct students. That window is a sample for short AI context, not full class history or a score.

## Rebuild and rollout

Review [the index file](../../firestore.indexes.json) before a hosted rollout. Local emulators do not prove production index readiness. Deployments and shared-data changes are separate operations.

1. Review and provision the exact indexes in `firestore.indexes.json` under an approved cloud scope. Wait until they are ready. A missing-index error returns 503 `query_index_required` with a clear operator action; it does not silently fall back to a full scan. Then use compatible dual-write API/workers before switching clients. Keep older writers stopped during a rebuild.
2. Inspect one page; this is read only:

   ```sh
   cd api
   go run ./cmd/rebuild-class-evidence --project YOUR_PROJECT --class CLASS_ID --limit 25
   ```

3. After review, `--apply` writes that page. Start with no `--after`; then pass the returned `next_cursor` to each next page. Keep the same class and limit. Each page scans at most 100 source rows. `candidates` counts eligible rows, not guaranteed writes: stale rows may be skipped.
4. The last applied page marks the generation complete. Check source counts, freshness, ownership, and a sample of current/older students before retiring compatible clients. Do not mark a partial data set as a full migration.
5. On failure, retry the same page or restart from an empty cursor. Starting again creates a new generation. Stable per-student writes preserve newer source evidence. Do not run concurrent operators.

To roll back reads, restore a compatible application version and keep source attempts. The projection can be rebuilt. It must follow the same retention, deletion, and tenant access policy as its source.

## Local evidence

The opt-in [query evaluation](../../api/cmd/evalqueries/main.go) uses 120 synthetic students and 1,440 attempts. A full fixture scan returns 1,560 rows. The default paged overview returns 51 query documents plus fixed ownership/synthesis reads. A student's 12 attempts remain reachable through 5-item pages. A 40-student evidence projection read returns 80 query documents plus class/manifest reads.

The report includes local duration, returned query documents, and point-document responses. These are controlled local measurements, not a production latency target, billing estimate, or index-entry scan count. See [evaluation commands](../../evals/README.md).
