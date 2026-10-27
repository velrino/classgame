# Release validation — October 27, 2026

The [release PR](https://github.com/velrino/classgame/pull/16) records integration and CI. A merge does not deploy the app.

| Check | Observed result |
| --- | --- |
| API | Existing Go tests, vet, and build pass |
| Web | TypeScript, lint, and production build pass |
| API policies | 49 synthetic cases pass through production Go rules |
| Browser evidence policies | 30 synthetic cases pass through production TypeScript rules |
| Firestore transactions | 10 local emulator cases pass, including concurrent claims, durable facts, replay, stale writes, resume, and bounded recovery |
| Queue delivery | Local synthetic Redis checks pass; see the queue report linked in the PR |
| Docs and assets | Local links, UPPERCASE Markdown names, folder indexes, exact Product tour, and original SVG pass |

The evaluation runs make no real model calls. Firestore checks use a loopback emulator, a `demo-` project, and synthetic records. Queue checks use a temporary local Redis. Shared Firebase data, rules, storage permissions, and cloud services are not changed.

Local entry pages and Scalar are checked separately from an authenticated teacher/student flow. The browser policy suite does not run real IndexedDB transactions across tabs. Live teaching quality, provider failover, production worker restart, and full browser recovery still need checks in an approved test environment.

## Open operational work

- [Private attachments](https://github.com/velrino/classgame/issues/5): a read-only audit found 8 public object ACLs and 4 unverified checks. Source changes do not revoke old access.
- [Student data access](https://github.com/velrino/classgame/issues/7): source API projections and rules need a reviewed live rollout.
- [Query scale](https://github.com/velrino/classgame/issues/10): class and mission pages are bounded; full classroom history and projection costs need work.
- [Shared AI budgets](https://github.com/velrino/classgame/issues/33): event claims are shared, while provider concurrency limits remain per process. Unknown provider outcomes do not allow exactly-once billing claims.
- [Retention and restore](https://github.com/velrino/classgame/issues/35): owners, keep times, deletion, backups, and restore targets need a separate decision.

No GCP deployment, new hosted worker, cloud permission change, shared-data bootstrap, or data deletion is included.
