# Operations

Review the intended environment before operational writes. Local startup does not recreate the former hosted application.

| Guide | Status and purpose |
| --- | --- |
| [Query scale](QUERY-SCALE.md) | Student pages, history, evidence freshness, and rebuilds |
| [Queues](QUEUES.md) | Dedicated Redis, worker roles, failed work and recovery |
| [AI budgets](AI-BUDGETS.md) | Shared provider admission, settings, and failure behavior |
| [Cloud Run](CLOUD-RUN.md) | Deployment inputs, identities, workers, and verification |
| [Hosting history](HOSTING-HISTORY.md) | Dated shutdown record; former URLs are historical |
| [Release safety](RELEASE-SAFETY.md) | Compatibility, recovery, and rollout checks |
| [Retention and recovery](RETENTION-RECOVERY.md) | Manual policy, encrypted scope export, and guarded restore |
| [Durable evidence](DURABLE-EVIDENCE.md) | Replay, claims, finalizer fences, and recovery |
| [Release validation](RELEASE-VALIDATION.md) | Dated local evidence and remaining checks |

Use the [release workflow](../processes/RELEASE-WORKFLOW.md) for Git integration and [validation](../development/VALIDATION.md) for local checks. A merged release does not establish a deployed service.
