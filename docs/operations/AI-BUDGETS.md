# Shared AI budgets

Every API and worker uses one shared Redis budget before calling the model. Teacher previews, tutor turns, learning traces, and class summaries use the same provider client. The local limit of 4 running calls and 16 admitted callers still applies per process.

## Settings

Use the same `REDIS_URL` database, scope, and limits for every process that shares a provider project or API key. Keep the scope technical, without student data. Different Redis databases or scopes create separate budgets.

| Setting | Safety default |
| --- | --- |
| `GENAI_SHARED_BUDGET_SCOPE` | `classgame-provider` |
| `GENAI_SHARED_CONCURRENT` | 4 running provider calls across all processes |
| `GENAI_SHARED_REQUESTS_PER_MINUTE` | 60 accepted calls per fixed minute |
| `GENAI_SHARED_OUTPUT_TOKENS_PER_MINUTE` | 65,536 reserved maximum output tokens per fixed minute |
| `GENAI_SHARED_LEASE_SECONDS` | 150; at least the configured request timeout plus 5 seconds |

These are conservative safety settings, not measured load targets or a currency budget. The repository owner owns changes. Use measured traffic, fallback rate, and provider quotas before raising them. Fixed minutes use Redis server time; two adjacent windows can have a burst. Input and thinking charges are not counted as an exact price by this guard.

The client reserves the call's maximum output before provider work. Release frees only its owned concurrent lease. Request and token usage stay until the window ends, even after an error or cancellation. A lost response may still have a provider charge.

## Failure and recovery

The guard fails closed when Redis is down, capacity is full, or instance settings differ. Existing model workflows select their deterministic fallback; the guard never starts paid work after a failed admission. Calls do not wait in a new queue. Redis operations have a one-second context limit and short I/O timeouts.

A crashed process keeps capacity until its lease expires. Unique owner tokens make a late or repeated release safe. Redis server time avoids API host clock differences. A lease cannot stop work inside the provider after a network partition, so this does not promise exactly-once billing or an absolute spending cap.

All instances must use the same settings. A changed policy is rejected while the old policy key is live. Drain old model work, apply matching settings, and let the old key expire after twice the larger lease/window duration of inactivity (5 minutes with defaults). Do not change the scope or clear keys to bypass a budget.

Budget keys use `classgame:provider_budget:{scope_hash}:*`. Use private Redis access, TLS/auth, a key-scoped ACL, and `noeviction` when these limits are required. Eviction, data loss, or an old Redis restore can reset usage. Durable student event claims remain in Firestore; this technical budget does not replace business idempotency. No Redis policy or cloud resource is changed by this source release.

## Observe

From `api/`, with the normal application environment configured:

```sh
go run ./cmd/aibudget
```

This command reads current active leases, accepted requests, and reserved output tokens. It starts no API, database, worker, or model request. It changes no budget counters. Logs use `genai shared admission rejected` and `genai shared lease release failed`, without prompts, credentials, or student text.

Alert on unavailable/mismatched settings and repeated release failures. Track full-budget rejections, active capacity, window usage, and fallback rate. Investigate sustained rejection before increasing limits; do not silently disable the shared guard.

## Local verification

```sh
python3 evals/platform/provider-budget/run.py --run
```

The opt-in runner creates its own temporary loopback Redis, checks its process ID, and stops every owned process. Eleven cases cover shared capacity, nonrefunded requests/tokens, stale owner release, lease expiry, matching settings, error/cancellation cleanup, two separate Go processes, and a stopped broker. It runs no real provider or database calls. Production partitions, provider-side cancellation, Redis failover, persistence, and restore still need environment checks.
