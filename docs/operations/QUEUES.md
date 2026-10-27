# Background queues

Use a dedicated Redis service for jobs. Never point jobs at disposable cache/rate-limit storage. This guide describes local commands and release gates; it does not provision or deploy services.

## Start

Set `TASKS_ENABLED=true`, `TASKS_REDIS_URL`, and `TASKS_QUEUE` in the server environment. The task URL has no cache fallback. Production requires `rediss://` with authentication. Keep credentials in secret storage, outside command arguments and logs.

Run the same binary in two roles, with the same application configuration:

```sh
# In api/, after setting the dedicated broker and required application settings.
APP_ROLE=api go run ./cmd/api
APP_ROLE=worker go run ./cmd/api
```

`APP_ROLE=all` is useful for local integration. `TASKS_CONCURRENCY` defaults to 4 and accepts 1–32 per worker process. A worker does not serve HTTP; use a long-running process/container with signal handling, restart policy and process health monitoring. A scale-to-zero HTTP service alone cannot provide continuous scheduled delivery. Never queue interactive tutor replies.

## Broker contract

- Use a separate broker endpoint and ACL identity. TLS and authentication are required in production. Network access stays private to the API and workers.
- Enable [persistence](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/) and replicas/backups according to an agreed data-loss target. Test restore and failover; an acknowledged Redis write is not an absolute no-loss guarantee.
- Set [no eviction](https://redis.io/docs/latest/develop/reference/eviction/). Memory exhaustion must reject enqueue operations rather than silently remove jobs. Monitor memory, queue age, retries, archives, worker health and pending application markers.
- Use Redis compatible with the pinned Asynq version. Do not assume Redis Cluster script compatibility. Do not share keys or run `FLUSHDB` on a working broker.
- Job bodies contain scoped IDs and correlation only. Restrict reads as application data. Completed jobs remain for 24 hours; archived jobs remain until reviewed/deleted. Decide and enforce retention, deletion and backup expiry before production use.

## Inspect and recover

The operator command uses only `TASKS_REDIS_URL`; it does not load Firebase, copy files or call a model. Its default is read only:

```sh
cd api
go run ./cmd/jobs
go run ./cmd/jobs --id JOB_ID
# Explicit write: run the job's existing stored body again after review.
go run ./cmd/jobs --retry JOB_ID
```

Fix dependency/configuration failures before retrying. Invalid jobs remain archived for review. A terminal archive is not a successful domain result. A changed body under the same ID returns a conflict instead of silently accepting different work.

Learning pending markers are scanned in bounded pages by worker recovery. Active claims and current scope/version checks remain authoritative. After exhausted retries, use `go run ./cmd/learning-recovery --limit 25 --redrive` after reviewing the same read-only page to advance the dispatch generation; normal scans must retain the stable ID.

Attachment transfer markers live in immutable mission revisions. Automatic attachment scans are not installed. After broker loss, inspect the stored `attachment_transfer_status`, class, mission and revision, then explicitly re-enqueue the known revision:

```sh
go run ./cmd/jobs --enqueue-attachment-class CLASS_ID \
  --enqueue-attachment-mission MISSION_ID --enqueue-attachment-revision REVISION
```

This only schedules the existing processor. It never grants public access or changes worksheet data itself. For an archived ID, review it and use `--retry`; for a completed ID restored from older database state, coordinate the 24-hour ID retention before re-enqueueing. Back up the broker before any queue deletion or bulk replay.

## Release gates and rollback

1. Verify the broker's ACL, TLS, no-eviction setting, persistence, backup restore, capacity and worker restart behavior in the intended environment.
2. Deploy compatible workers before API producers. Do not mix incompatible job schemas. Verify a synthetic queue check, delayed delivery, shutdown/restart, backlog age and archived-job handling.
3. Reconcile old pending markers before changing queue routing. Old Cloud Tasks delivery endpoints no longer exist; do not leave an old producer sending to them. Keep old job inventory until reviewed completion or scoped re-enqueue.
4. Exercise real attachment and learning workflows only in an approved test environment. The local smoke proves transport behavior with synthetic handlers; it does not prove Firebase/model outcomes or production failover.
5. On failure, stop new producers, drain or pause workers and preserve the broker plus pending markers. Roll back API and worker versions together. `TASKS_ENABLED=false` restores inline learning behavior but does not deliver attachment jobs or inactivity watchdogs. Reconcile pending work after recovery; do not delete it to clear an alert.

Run `python3 evals/platform/queue-smoke/run.py` from the repository root for the opt-in local transport check. It verifies ownership of its temporary localhost Redis before sending jobs, refuses port 6379, and runs no domain database or real provider calls. Its 11 checks include two worker processes with concurrent producers, SIGKILL recovery through real lease expiry, and bounded enqueue failure after the actual broker stops. The crash check usually takes 1–2 minutes. It is separate from the default offline policy suite.

The local result does not verify network partitions, managed Redis failover, persistence/backup restore, max-memory/no-eviction behavior or loss of an acknowledgement after an external effect commits. Test these with approved staging infrastructure before enabling jobs in production. Domain idempotency/version guards remain required: a repeated delivery can still repeat an external effect or provider charge. Do not treat a synthetic counter that stays at one as an exactly-once guarantee.
