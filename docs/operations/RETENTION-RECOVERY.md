# Retention and recovery

Retention is manual by default. No TTL, automatic cloud deletion, hosted backup job, or live restore is enabled by this release. The class teacher owns class archive/deletion approval. The repository owner operates backups, key recovery, and access changes. A tenant delegate can approve work only for that tenant.

## Policy defaults

These are explicit starting settings for operator review, not legal or compliance claims. A lower keep time needs owner approval and a private audit record. Active references, legal holds, or unresolved failures block deletion.

| Data | Keep or review rule |
| --- | --- |
| Active classes, missions, immutable revisions, private worksheets | Keep while active or referenced by an attempt/job; no automatic expiry |
| Archived class records | Review 90 days after the teacher-approved archive date; deletion remains a separate approved action |
| Attempts, accepted facts, tutor event replay, checkpoints, private class projections | Keep with their class/attempt. Never expire replay events before the attempt or reset a generation count by deleting history |
| Global user profiles and memory, teacher/auth identities | Separate account review after 90 inactive days and no active references. A class cleanup never deletes a global identity or another class's memory |
| Unpublished/orphan uploads | Review after 7 days; prove ownership and absence of references before removing any file |
| Completed queue jobs | Existing 24-hour queue retention; durable application state remains authoritative |
| Failed/archived queue jobs | Review within 7 days. Keep until resolved or explicitly retired; no silent purge |
| Content-free operational logs | 30-day sink retention default, set by the operator; no student text, prompts, or tokens |
| Encrypted backups | Daily backup target and a fresh backup before approved data work; 30-day retention, private access, key stored separately |
| Minimal deletion/restore receipts | Keep until all related backups and job replay windows expire; 365-day review default. No student text in receipts |

Backup planning targets are RPO 24 hours and RTO 4 hours. They are targets to measure in the intended environment, not service guarantees. The local roundtrip proves data/claim behavior, not those production timings. Do not restore records that a later approved deletion register says must stay deleted.

## Read-only plan and encrypted class export

The operator needs Firestore read access and Application Default Credentials (or `GOOGLE_APPLICATION_CREDENTIALS`). The tool does not load API bootstrap, start workers, or call a model. Always pass all four scope IDs. From `api/`:

```sh
scope_flags=(-project PROJECT -organization ORG -district DISTRICT -school SCHOOL -class CLASS_ID)
go run ./cmd/classbackup "${scope_flags[@]}"
# Optional: add -archived-at 2026-01-01T00:00:00Z -archive-days 90 for a review date.
```

The default prints counts and review status, with zero database writes. Review eligibility does not authorize deletion. For export, choose owner-only paths outside Git, with the key separate from backup storage. Generate a 32-byte key with `umask 077; openssl rand -out KEY_FILE 32`, then:

```sh
go run ./cmd/classbackup "${scope_flags[@]}" -mode export \
  -file PRIVATE_BACKUP_FILE -key-file PRIVATE_KEY_FILE
```

Exports use authenticated AES-256-GCM encryption, owner-only files, exclusive creation, and a versioned checksum. Existing files are never overwritten. Raw student fields, encryption keys, and recovery owner tokens are not printed. Keys/backups inside Git, including paths through symlinks, are rejected.

A class bundle includes class-owned missions/revisions/code lookups, attempts/tutor event documents, completion/catalog projections, class summaries, and the private class-evidence parent/rows. Tenant/class scope is checked before a bundle is accepted. Global users, auth identities, tenant hierarchy, and stored file bytes are excluded. A class bundle is not a complete account erasure or object-storage backup.

The tool accepts at most 200 documents and 8 MiB per class bundle. It fails instead of producing a partial backup. Larger scopes need a reviewed managed database backup and a separate private object-storage backup. A managed Firestore export is a cloud operation; it is not run by local setup or these evaluations. Preserve immutable revisions, object generations, and the approved deletion register together.

## Recovery procedure

1. Record the owner, scope, archive approval, backup hash, object inventory, rollback point, and later deletion restrictions in a private audit record.
2. Stop **all** API/worker producers for the affected data. Drain or pause queues and in-flight provider work. A recovery lease coordinates recovery commands; it does not automatically stop application writers.
3. Export after the write hold. Verify key recovery, scope, counts, typed data, objects, and backup completeness. A live multi-read export is not a point-in-time snapshot while writers continue.
4. Use a separately reviewed environment-specific procedure for hosted changes. The repository's delete/restore modes refuse hosted projects: they require a numeric loopback emulator, a `demo-` project, and `-apply-local`.
5. Restore into an empty scope. Current-version checks make local deletion atomic; create-only checks prevent restore from overwriting current rows. A checksum/owner receipt makes repeat restore a no-op after completion.
6. Restore clears old tutor/finalizer lease owners and advances finalization dispatch generation. It preserves accepted facts, counts, completed event replay, Firestore timestamps, and integer values. Review old queue jobs before restarting; an unknown provider charge is never refunded by restore.
7. While writers are still stopped, run the read-only [class evidence rebuild plan](QUERY-SCALE.md), apply reviewed pages explicitly, reconcile private projections/files, and check student-safe access. Refresh old browser sessions before reopening writes. Account/global-memory and file cleanup require their own scoped review.

Local-only recovery commands use the same scope, encrypted artifact, and key:

```sh
go run ./cmd/classbackup "${scope_flags[@]}" -mode delete-local -apply-local \
  -file PRIVATE_BACKUP_FILE -key-file PRIVATE_KEY_FILE
go run ./cmd/classbackup "${scope_flags[@]}" -mode restore-local -apply-local \
  -file PRIVATE_BACKUP_FILE -key-file PRIVATE_KEY_FILE
# Abort a held plan before any deletion, using its original bundle:
go run ./cmd/classbackup "${scope_flags[@]}" -mode release-local -apply-local \
  -file PRIVATE_BACKUP_FILE -key-file PRIVATE_KEY_FILE
```

A changed source, another recovery owner, expired hold, nonempty restore target, or altered artifact fails closed. After deletion, only the same bundle/owner may resume restoration. Do not clear recovery receipts or queue state to force a retry. A restore receipt contains technical scope/hash/owner state and stays private under the default deny browser rules.

## Local evidence

```sh
FIRESTORE_EMULATOR_HOST=127.0.0.1:8787 python3 evals/platform/retention/run.py --run
```

The runner compiles the real commands, uses a unique synthetic fixture in `demo-classgame-retention`, and removes only its own rows/files. Fourteen cases cover read-only plan/export, authenticated encryption, scope/duplicate rejection, owner holds, atomic version checks, other-class/tenant/global-user isolation, typed/projection restore, stale worker fences, preserved event replay, and idempotent restore. There are no real provider calls or shared cloud writes.
