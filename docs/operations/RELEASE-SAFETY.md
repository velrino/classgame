# Release safety and recovery

Use an integrated build and development data to verify behavior. A draft PR, passing compilation, or healthy dependency response does not establish a safe rollout.

## Runtime bounds and correlation

The shared GenAI client admits at most 16 callers and 4 provider calls per API process. Excess work fails immediately into existing workflow fallback handling; no generation queue is introduced. Limits are reviewed code constants.

Text instructions, prompt, and response schema share a 256 KiB input budget. Inline data is bounded to 5 parts and 20 MiB total. Oversize inputs are rejected, not silently truncated. Output tokens are capped by both configured maximum and 8,192; decoded response text is bounded to 256 KiB. The default timeout remains 30 seconds, with a 2-minute ceiling and earlier caller cancellation honored.

Identical in-flight inputs may coalesce inside the same client and hashed caller-token scope. Completed results are not cached; each caller decodes its own JSON target. The first caller's cancellation/deadline governs shared provider work; other waiters can cancel independently. Admission remains held until retained shared-result channels are released, bounding canceled waiters. This is process-local duplicate suppression, not durable idempotency or exactly-once execution across replicas.

The API validates or replaces `X-Request-ID` and uses standard OpenTelemetry propagation APIs to accept/return W3C `traceparent`. Cloud Tasks inherits request/trace correlation and workers create a child correlation context. These headers do not authenticate requests; OIDC validation remains authoritative for workers. Request/task logs contain identifiers, route templates, status, and duration, without request bodies, bearer values, or worksheet/student content. No tracing exporter or SDK is installed by this propagation-only integration.

## Compatibility and migration order

1. Inventory old clients, checkpoint shapes, published revision references, object access, and current rules. Record rollback artifacts and migration checkpoints before data changes.
2. Expand the API with safe class reads and authorized attachment delivery. Preserve immutable revisions and tolerant checkpoint handling; never trust an older client's optimistic completion flag as evidence.
3. Deploy compatible web consumers. Confirm pending answer events reach the server before a final checkpoint and that recovery works across reloads and a newer publication.
4. Backfill any new projection, verify its completeness, then switch readers. Restrict old administrative reads only after compatible consumers are ready; document how stale clients are refreshed or rejected.
5. Contract old representations only after the measured compatibility window. Do not delete source records needed by active attempts or delayed jobs.

Private attachments require compatible bearer-authenticated download handling. If an old client cannot use it, refresh/block that version or provide a separately authorized compatibility path; restoring public object access is not a rollback strategy. API changes to new objects do not revoke existing public ACLs: inventory and remediate those separately, checking bucket policy as well as object ACLs. A read-only inventory is not remediation.

## Release gates

- Existing Go and web checks pass on the exact integration commit; dependency advisories have current reachability and remediation decisions.
- Teacher/student workflows pass, including revision conflicts, incorrect answers, checkpoint recovery, older checkpoints, pending answers, and private downloads.
- Negative checks reject wrong providers, unrelated classroom access, forged completion, invalid worker identity, and unsafe object access. Verify deployed Firebase/Storage rules separately from local checks.
- Concurrency saturation, provider timeout, and caller cancellation select an honest bounded fallback. Duplicate requests/jobs preserve accepted progress; trace/request correlation survives API-to-worker delivery.
- Projection backfill, source/consumer compatibility, retry handling, and rollback steps are recorded. Original Product tour, architecture SVG, and runtime asset URLs remain intact.

Record measured latency/read counts, fallback rates, and remaining delivery gaps. Introduce an outbox, breaker, or cache only when a measured requirement justifies its consistency and recovery costs.

## Retention and deletion decisions

Before production use, assign an owner and explicit retention/deletion rules for uploads, immutable mission revisions, checkpoints/events, tutor output, student memory, class summaries, logs, and backups. Record purpose, access, retention window, and deletion behavior rather than inventing a universal duration.

A deletion procedure must cover derived projections and stored objects, prevent delayed jobs from recreating deleted data, and state what retained history/backups still contain. These decisions and existing public-object remediation remain operational work; this release does not claim they have run.

## Rollback and recovery

Retain a compatible application artifact and configuration snapshot. On failure, stop affected writes, preserve checkpoints/evidence, and restore the compatible version while retaining authorization and private-storage controls. Check active revision pins and pending jobs before resuming.

Resume interrupted backfills from a recorded cursor and reconcile their projection counts. Reuse stable job identities; distinguish retryable outages from permanent invalid input. When a provider times out, do not report successful generation or blindly replay unrelated business effects. Recheck health and the affected authenticated workflow after recovery.
