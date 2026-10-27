# Task portability

Background attachment copying and learning jobs now use Go workers with [Asynq v0.26.0](https://github.com/hibiken/asynq/tree/v0.26.0) and a dedicated Redis broker. The Cloud Tasks adapter, SDK dependency, worker HTTP endpoints and OIDC delivery configuration have been removed. No hosted resources were changed.

[Message](../../api/internal/taskqueue/message.go) carries a stable ID, JSON payload, correlation metadata and delivery time. Module-owned schedulers keep job meaning and IDs. [The Redis adapter](../../api/internal/platform/redisqueue/client.go) owns delivery, delay, retries and archived work. [Application wiring](../../api/internal/app/background.go) connects it to the existing domain processors; it does not copy their policies.

`TASKS_ENABLED=false` remains the local default. Learning finalization uses its inline fallback; attachment copying and scheduled inactivity jobs need a worker. `APP_ROLE=api` serves HTTP, `worker` processes jobs, and `all` does both using the same manual setup. Interactive Mia remains in its synchronous request path.

## Delivery limits

Delivery is at least once, without ordering guarantees. Stable IDs reject a changed payload while the ID exists; identical payloads share the existing job. Completed IDs are retained for 24 hours. This is a bounded replay window, not permanent uniqueness or exactly-once model charging. Retries reuse the stored body. Invalid jobs are archived; transient failures retry up to ten times, with a five-minute task deadline. Worker shutdown allows 30 seconds before unfinished work is returned for recovery.

Attachment publication enqueues before its transaction and records the immutable revision's transfer marker. A failed enqueue prevents publication. If publication is still committing, the worker retries. Completed object copies and obsolete revisions remain safe to repeat through the existing processor. A lost Redis acknowledgement may leave an extra job; it must not justify inventing new business IDs. Broker data loss still needs a reviewed re-enqueue from the persisted revision marker.

Learning jobs use persisted pending markers and scoped finalization claims. The recovery sweep retries normal pending work with the same job identity. An explicit operator redrive is needed after an archived/exhausted job; it must use the domain recovery command rather than changing the JSON in Redis. See [queue operations](../operations/QUEUES.md) for recovery and deployment gates.

## Hosting boundary

Asynq is still v0.x: its API can change before 1.0, and the upstream project warns that some Lua scripts are incompatible with Redis Cluster. The pinned implementation targets a standalone-compatible endpoint; validate any managed service before adoption. Use a separate broker, persistence, no eviction, backups and tested restore/failover. Redis is now infrastructure the team must operate.

This change does not move Firebase Auth, Firestore, object storage or the AI provider. Provider admission is shared across API and worker processes through Redis; only matching-call sharing remains per process. Multiple replicas may still incur duplicate model charges after an unknown outcome. Application version checks and leases protect stored outcomes, not global cost uniqueness.
