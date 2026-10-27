# Operations

Review the intended environment before operational writes. Local startup does not recreate the former hosted application.

| Guide | Status and purpose |
| --- | --- |
| [Cloud Run](cloud-run.md) | Deployment inputs, identities, workers, and verification |
| [Hosting history](hosting-history.md) | Dated shutdown record; former URLs are historical |

Use the [release workflow](../processes/release-workflow.md) for Git integration and [validation](../development/validation.md) for local checks. A merged release does not establish a deployed service.

- [Release safety](release-safety.md): compatibility, migration order, rollback, recovery, and validation gates.
