# Task portability

The queue message is now independent of Google Cloud. Cloud Tasks is still the only implemented delivery backend. This change does not install Redis workers, change the backend, or move hosted resources.

## Current boundary

[Message](../../api/internal/taskqueue/message.go) holds a stable ID, JSON payload, metadata and delivery time. Workflow-owned ports accept this type. The attachment and learning schedulers no longer import the Cloud Tasks adapter. Routing, HTTP delivery, OIDC and provider limits stay in [the adapter](../../api/internal/platform/cloudtasks/client.go).

Local development currently uses `CLOUD_TASKS_ENABLED=false`. Learning finalization can use the existing inline fallback. Background attachment copying and scheduled watchdog delivery are not exercised in this mode. Interactive Mia replies stay in the request/stream (SSE) path, with deadlines and cancellation; they are not offline queue jobs.

## Backend choice

| Option | Fit for this Go API | Operations work |
| --- | --- | --- |
| [Cloud Tasks](https://docs.cloud.google.com/tasks/docs/dual-overview) | Managed HTTP delivery while hosting on GCP | Configure queue limits, worker identity, retries and recovery. Google operates the queue. |
| [Asynq with Redis](https://github.com/hibiken/asynq) | First candidate if we later choose Go workers across clouds | Run Go workers and a durable Redis service. Own capacity, failover, backups and recovery. |
| [BullMQ](https://docs.bullmq.io/) | Its main library uses Node.js; useful if Node workers are already a team choice | Another worker runtime for this Go application, plus Redis operations. |

Asynq provides scheduling, retries and worker-crash recovery in Go. Its documented Redis Cluster limits need a compatibility check before choosing a managed Redis service. This is a candidate, not an installed dependency or a completed migration.

Changing the queue alone does not move Firebase Auth, Firestore, object storage, Vertex AI or worker identity. An AWS/Azure plan must decide which adapters to keep or replace, then verify authorization, data mapping, deadlines and recovery.

## Reliability across instances

[Cloud Tasks can deliver duplicates and does not promise execution order](https://docs.cloud.google.com/tasks/docs/common-pitfalls). A future Redis worker must also handle repeated or late delivery. Keep stable job IDs, pinned revision/attempt scope and transactional outcome checks. A retry must not apply old work to the student's new session. Idempotency means repeating a job keeps the same stored result.

The AI request limit and matching-call sharing (coalescing) work inside one API process. Some request rate limits already use Redis, but there is no shared provider cost/concurrency budget. Two API instances can still call the model for the same event. Persisted tutor-turn replay protects stored writes; it does not prevent duplicate provider charges.

Future multi-instance work should cover:

- A shared provider budget and bounded worker concurrency, measured across replicas.
- A durable operation lease for costly duplicate work when needed. A lease has an owner and expiry. A version token, checked on result writes, prevents an old worker from overwriting a newer result. A crash after a provider charge can still cause another charge unless the provider offers a suitable idempotency/replay contract.
- A durable job-intent/outbox when one database transaction must publish state and schedule recoverable work. A dispatcher retries delivery after a crash.
- A recovery scan for missing pending jobs, expired leases and failed jobs. Set retry limits, backoff, alerts and a failed-job archive/dead-letter queue with reviewed replay.

A future Redis queue needs a dedicated broker, separate from disposable cache/rate-limit data. Choose [persistence and tested restore/failover](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/), with an explicit data-loss target. Use [no eviction](https://redis.io/docs/latest/develop/reference/eviction/) and handle enqueue errors when memory is full. Persistence and replication do not replace application recovery or scoped idempotency.

These controls are follow-up design work. The implemented change closes the message-type coupling while preserving current job payloads, IDs, scheduling, OIDC and retry behavior.
