# Release validation — October 27, 2026

[Release PR #54](https://github.com/velrino/classgame/pull/54) follows the modular source release in [PR #16](https://github.com/velrino/classgame/pull/16). Git integration does not deploy the app.

| Check | Observed result |
| --- | --- |
| API and web | Existing Go tests/vet/build and web typecheck/lint/build pass; final integrated CI is recorded in the PR |
| API policy | 66 synthetic cases through production Go rules, including teacher eligibility, proxy trust, and enrollment |
| Frontend policy | 30 evidence cases and 13 reducer/recovery cases through production TypeScript |
| Real browser storage | 14 native IndexedDB cases, including two tabs, scoped ACKs, leases, capacity, and reload |
| Real local app | Teacher login, class creation, preview SSE, publication, anonymous join, wrong answer, reasoning, feedback, offline reload, Retry, and server-confirmed completion |
| Firestore transactions | 10 real local claim/fact/replay/finalizer cases |
| Query scale | 12 cases with 120 students and 1,440 attempts; stable roster/history pages and private projection rebuilds |
| Queue delivery | 11 real local Redis checks, including two workers, worker kill/recovery, and stopped broker |
| Shared provider budget | 11 real local Redis cases plus 7 refusal guards; two processes and store outage |
| Private upload adapter | 17 synthetic SDK cases plus 4 refusal guards; no real network call |
| Retention and restore | 14 real local recovery cases plus 9 refusal guards; typed data, isolation, fences, and repeat restore |
| Docs and assets | Links, UPPERCASE Markdown, folder indexes, exact Product tour, and original architecture SVG pass |

The browser review found and fixed incorrect advancement after a wrong answer, a crash on nullable server checkpoints, stale async updates, and delivery copy that appeared before acknowledgment. See [local flow evidence](../development/LOCAL-EMULATORS.md). Provider-disabled fallback is explicit; teachers must review whether a generated or fallback preview matches the lesson.

## Measured local baseline

The query fixture's unbounded comparison read 1,560 documents in 56 ms. A default student page read 51 query documents and 2 point documents in 35 ms. A projected 40-student page read 80 query documents and 2 point documents in 103 ms. Complete roster and history paging had no gaps. These are local emulator observations, not production latency guarantees.

The real local HTTP run recorded a median request time of 5.75 ms. The shared provider-budget outage case rejected work in 406 ms without calling a provider. The two-process budget case observed peak2 at limit2, with2 accepted and14 denied calls. No real model call was made. Local fallback checks do not establish live teaching quality, provider billing, or managed failover behavior.

## Approved cloud access review

A separate user approval allowed only security access restrictions. The active Firestore rules now make teacher administrative class documents teacher-only, with an exact source readback. Eight reviewed public object ACLs were removed. Public Access Prevention is enforced on the verified worksheet bucket. One worksheet token was removed with generation/metageneration conditions; other metadata and file generation were preserved. Final checks found zero worksheet download tokens and denied anonymous generation-specific reads on all15 worksheets and the empty marker.

The current API identity can read8/12 referenced files privately. Four legacy files return403 and need their existing owner to approve narrow private runtime access. No new grant was allowed or applied. [Issue #5](https://github.com/velrino/classgame/issues/5) stays open for this concrete blocker. Keep these files private.

No app deployment, new hosted service/worker, file deletion, Firestore data migration, or shared-data bootstrap was run. There is no hosted app or worker to validate. Hosted index setup, proxy paths, Redis TLS/ACL/no-eviction, managed failover, and production recovery remain explicit gates before any future deployment. The [retention guide](RETENTION-RECOVERY.md) defines manual policy and bounded local recovery; it does not claim a measured production RPO/RTO.
