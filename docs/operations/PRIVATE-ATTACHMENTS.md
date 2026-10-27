# Review old attachment access

New downloads use the authenticated API. Old public object ACLs need their own review; a private writer does not revoke old access.

The legacy audit command is read-only by default. Create a local review manifest with the explicit application project, bucket, and credential file:

```sh
cd api
go run ./cmd/backfill-mission-attachment-storage-urls \
  --project PROJECT --bucket BUCKET --manifest-out /private/local/access-plan.json
```

The manifest contains private object paths. It is created with mode `0600`, never overwritten, and must stay out of Git/public issues. It records generation, metageneration, ACL hash, and only verified public-reader actions. Unverified objects and download-token metadata block application. Token removal needs its own reviewed, generation-fenced metadata change. The current account needs `storage.objects.getIamPolicy` to inspect private object access and `storage.objects.setIamPolicy` to change it.

After review and explicit authorization, the separate apply command is:

```sh
go run ./cmd/backfill-mission-attachment-storage-urls \
  --project PROJECT --bucket BUCKET --apply-manifest /private/local/access-plan.json \
  --confirm-project PROJECT --revoke-public
```

Apply validates every object before any update, rejects old or changed manifests, preserves all nonpublic ACL entries, and uses atomic object generation/metageneration conditions. It never deletes files, grants public access, rewrites Firestore, or changes bucket IAM. A repeated run safely accepts already-private objects. After a partial failure, keep the private objects private and create a new read-only plan.

Verify authenticated reads still work and anonymous reads fail. Bucket IAM must also be private. A public bucket binding requires a separate reviewed change; this tool will stop.

## Approved live access review

The separate security-access approval was applied on 27 October 2026. Eight verified public object ACLs were removed with generation/metageneration conditions. Four legacy object ACLs could not be inspected or changed with the current account. Bucket Public Access Prevention is now enforced and overrides their public principals. The bucket inventory contained only 15 ClassGame worksheets and one empty folder marker. Generation-specific anonymous origin checks returned `403` for all 16 objects.

One worksheet had Firebase download-token metadata. Only that token key was removed with generation/metageneration conditions; other metadata and the file generation were preserved. This is separate from ordinary public ACL checks. Do not assume Public Access Prevention revokes Firebase token links or old cached copies.

## Private runtime read repair

The later approved follow-up added one `roles/storage.objectViewer` binding for the existing local API account. Its condition matches only the object resource type and four exact worksheet paths. The change uses project IAM because conditional bucket bindings require uniform access; the current fine-grained bucket access stays unchanged. See [Google's condition rules](https://docs.cloud.google.com/storage/docs/access-control/iam#conditions).

The policy was read and written as version 3 with the original `etag`. All nine old bindings were preserved. The exact readback matched the planned change. No new identity, bucket-wide read grant, object write/delete permission, or public access was added.

After the change, all 12 generation-pinned reads and all 12 full reads through the production attachment adapter passed. Anonymous generation-specific requests still returned `403` for all 15 worksheets and the empty marker. Worksheet download tokens remain zero, and the four reviewed files kept the same generation and metadata. No file content was printed or saved.

The grant follows four exact logical paths, including other generations or future replacements at those names. IAM cannot pin a generation. Keep attachment names immutable and review this binding when the configured runtime identity changes. The four legacy ACLs remain unreadable and unchanged; Public Access Prevention blocks their public entries. New-upload readiness still fails closed when private access cannot be proven.

To roll back this grant, remove only the `ClassGameLegacyWorksheetReads` binding using a fresh version-3 policy and `etag`. Preserve other bindings and audit settings. Recheck private reads and anonymous denial. Do not replace a newer policy with an old snapshot or restore public access.

The active Firestore rule now makes administrative class documents teacher-only; the exact active source was read back and matched the repository. Local compatible-client checks cover teacher reads, denied student admin reads, denied browser writes, and enrolled mission subscriptions.

No app deployment, new service, file deletion, or Firestore data write ran. The only new grant is the read-only condition described above. Private manifests and aggregate evidence stay outside Git.
