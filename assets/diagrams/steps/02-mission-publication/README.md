# 02 — Review and publish the mission

[Open the full PNG](diagram.png) · [Mermaid source](SEQUENCE.txt)

Start with the saved preview from [step 01](../01-mission-draft/README.md). The teacher reviews it and sends `expected_draft_revision` with optional edits. The API checks access and content policy, prepares any file-copy job, and publishes through a Firestore transaction. The result is a published revision; file copies may still be pending.

## Important boundaries

- A configured Redis queue accepts the revision-scoped copy job before the Firestore transaction. These are separate operations. A queue failure stops publication.
- With jobs disabled, publication can continue without a background copy.
- A worker can receive the job before publication. It retries while the expected revision is pending. Worker execution belongs to step 03.
- The transaction rechecks ownership and revision. Publishing unchanged content without inline edits reuses the revision, while refreshing class activation and the student catalog.

## Firestore records

| Record | Publication behavior |
| --- | --- |
| `missions/{mission_id}` | Save publication state, snapshot and `published_revision`. |
| `missions/{mission_id}/published_revisions/{revision}` | Store the published content snapshot. |
| `teacher_classes/{class_id}` | Set `active_mission_id`; increment `mission_count` only on first publication. |
| `student_class_missions/{class_id}/missions/{mission_id}` | Write the safe catalog for all-student missions; remove it for selected audiences. |
| `mission_codes/{mission_code}` | Update the optional sharing-code lookup. |

## Failure paths

| Response | Meaning |
| --- | --- |
| `400 validation_failed` | Missing or invalid expected revision or edited fields. |
| `403 forbidden` | Teacher or class access is denied. |
| `404 not_found` / `410 mission_deleted` | The mission is missing or deleted. |
| `409 draft_conflict` | Draft revision changed, including a race before transaction commit. |
| `422 mission_guardrail_blocked` | The stored content guardrail blocks publication. |
| `503 attachment_queue_unavailable` | The configured queue did not accept the job. No publication transaction starts. |

The diagram shows the main flow and key branches. It is not an exhaustive HTTP error map.

## Known integrity limit

Inline approval edits currently do not advance the draft revision. Different edited approvals can overwrite the same published revision. This is tracked in [issue #81](https://github.com/velrino/classgame/issues/81); this diagram does not claim immutable snapshots or general exactly-once publication.

The `create participant` statements introduce existing services visually. They do not provision services. Redis is optional, so its header marks the first possible queue call.

Sources: [approval service](../../../../api/internal/modules/missions/service.go), [publication transaction](../../../../api/internal/modules/missions/firestore_repository.go), [HTTP responses](../../../../api/internal/modules/missions/handler.go), and [revision-pending worker](../../../../api/internal/modules/missions/attachment_transfer.go).

## Regenerate

From the repository root:

```sh
npx --yes --package @mermaid-js/mermaid-cli@11.17.0 mmdc \
  -i assets/diagrams/steps/02-mission-publication/SEQUENCE.txt \
  -o assets/diagrams/steps/02-mission-publication/diagram.png \
  -b white -w 2200 -s 1.5
```

Use a white preview canvas and PNG export background.
