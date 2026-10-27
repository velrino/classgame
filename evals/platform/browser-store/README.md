# Real browser storage evaluation

This opt-in local page imports the production IndexedDB adapter and queue policy. Node.js 24 strips TypeScript types; no package, business-rule copy, Firebase client, or real provider is added.

From the repository root:

```sh
node evals/platform/browser-store/run.mjs --run --port 3102
```

Open the printed numeric loopback URL. Select **Run browser storage checks** and allow its local peer tab. Keep the tabs in the same browser profile. After the storage checks, select **Check reload recovery**. The page reloads and reads the original saved event before acknowledging it. The peer closes after the final report.

The fourteen cases use native IndexedDB, real transactions, two independent tab contexts, and production functions. They check concurrent writes without lost rows, exact duplicates, changed intent, actor/class/mission/revision/attempt isolation, capacity refusal without eviction, one live lease, old-owner fencing, exact scoped ACKs, transient and permanent retries, field projection, large-record refusal, and real reload recovery.

Lease transfer uses the production function's explicit clock argument; it does not wait for wall-clock expiry.

Only generated `eval-browser-<UUID>` actor records are removed. The page runs on its own origin, separate from the product. It never resets the whole browser database. A stopped run can use **Clean this run's synthetic work**. The production queue has no silent expiry.

A new immutable folder under `evals/output/` contains the report, source revision/dirty state, production-module hash, selected cases, and zero provider/shared-database calls. The server accepts reports only from its own origin with a generated per-process token, serves a fixed file list, and binds only to loopback. It refuses implicit startup and product/default service ports.

This verifies the storage adapter in a real browser. It does not by itself verify React recovery, full device offline behavior, storage eviction, Firebase rules, or model quality. Use the [local app workflow](../../../docs/development/LOCAL-EMULATORS.md) for those app flows and the local HTTP suite for server boundaries.
