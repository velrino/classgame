---
name: classgame-background-jobs
description: Inspect or change ClassGame background dispatch, retries, worker recovery and queue adapters while preserving durable learning and attachment state.
---

# ClassGame Background Jobs

Read [task portability](../../../docs/architecture/TASK-PORTABILITY.md) and the relevant part of [queue operations](../../../docs/operations/QUEUES.md). Follow module-owned schedulers and processors through application wiring before editing transport code.

Keep business meaning and stable task IDs in their consuming modules. The queue adapter owns delivery mechanics. Use dedicated `TASKS_REDIS_URL`; never fall back to quota/cache `REDIS_URL`. Firestore markers, claims and refresh intents remain durable business state.

Preserve these boundaries:

- Delivery may repeat or arrive out of order. Payload checks and 24-hour completed-ID retention do not guarantee permanent uniqueness, exactly-once effects or one model charge.
- Retain publication enqueue/transaction ordering, immutable revision identity, terminal attachment outcomes, owner/lease checks and source/dispatch generation fencing.
- Keep provider/queue calls outside retryable Firestore transactions. Recovery must not regrade students or invent new IDs after an uncertain enqueue.
- Mia tutoring stays synchronous. With jobs disabled, learning finalization keeps its inline fallback; attachment copying and inactivity jobs still need workers.

For adapter/schema changes, explain compatible worker-before-producer rollout and pending-work recovery. Redis portability does not migrate Firebase or AI dependencies. Review broker persistence/no-eviction/restore requirements rather than claiming no-loss delivery.

Use relevant existing checks within task scope; local smoke results do not prove production failover. Do not add tests without an explicit request. Do not run redrive, deletion, cloud changes or deployments without matching task authorization.
