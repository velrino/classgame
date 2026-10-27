# 03 — Copy mission files to private storage

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

Start with the copy job from [step 02](../02-mission-publication/README.md). The **Worker** runs the Asynq attachment processor. It loads source metadata from the published DB snapshot, copies remaining files from Uploadcare to **private Cloud Storage**, and saves each outcome. Action 14 completes the queue job; individual files may have permanent failures.

## Retry and failure rules

| Situation | Worker behavior |
| --- | --- |
| Published revision may still commit | Retry while it is pending. |
| Revision cannot exist or does not match | Finish without copying. Existing historical revisions, including soft-deleted mission revisions, can still run. |
| File is already stored or permanently failed | Skip it during preparation. |
| Valid object already exists | Reuse it after scope and private-access checks. Create-only writes prevent replacement; concurrent writers reread the winner. |
| Temporary copy or DB error | Return an error for retry. The job allows 10 retries and a 5-minute execution timeout. |
| Permanent file error | Save the failure and continue. Final state can be `complete_with_failures` while the queue job succeeds. |
| Invalid job or exhausted retries | Archive for operator review. Completed jobs stay for 24 hours. |

The job body contains only class, mission, and revision IDs. The copier checks **HTTPS source hosts and redirects**, limits new file streams to **20 MiB**, and uses a 3-minute download timeout. It verifies bucket IAM, object ACLs where used, and the absence of public download tokens. It checks access; it does not repair permissions. File delivery uses the authenticated API, with no public delivery URL created here.

## Stored records

| Record | Transfer behavior |
| --- | --- |
| `missions/{mission_id}/published_revisions/{revision}` | Track copying, attempts, per-file results, and the final transfer state. |
| `missions/{mission_id}` | Mirror attachment updates only into matching draft/published revisions. |
| `classes/{class}/missions/{mission}/revisions/{revision}/{source_key}/{file}` | Deterministic private object path. Store bucket, object path, MIME type, and size in the attachment result. |

Legacy class-nested mission records remain readable. Storage calls happen outside Firestore transactions. The `create participant` statements only place existing services at first use.

## Current limits

This flow has **no separate attachment owner token**. The transaction checks full attachment identity and preserves the first validated stored object and timestamps. Late failures cannot downgrade stored results or make a permanent failure retryable. A validated success already in flight may recover a failed file; other corrected source content uses a new published revision. These rules support retries without claiming exactly-once execution.

There is no automatic attachment recovery scan. After broker loss or archived work, follow the [queue recovery guide](../../../../docs/operations/QUEUES.md). Private-access setup is covered in [private uploads](../../../../docs/operations/PRIVATE-UPLOADS.md).

Sources: [processor](../../../../api/internal/modules/attachments/processor.go), [revision transactions](../../../../api/internal/modules/missions/attachment_transfer.go), [storage copier](../../../../api/internal/modules/attachments/storage.go), [queue settings](../../../../api/internal/platform/redisqueue/client.go), and [job outcomes](../../../../api/internal/app/background.go).

## Regenerate

From the repository root:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/03-attachment-transfer/SEQUENCE.txt \
  -o assets/diagrams/steps/03-attachment-transfer/diagram.png \
  -b white -w 2200 -s 1.5
```

Use a white preview canvas and PNG export background.
