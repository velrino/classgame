# Private uploads

The attachment copier checks access before it marks a file ready. It never repairs cloud permissions. Keep the bucket private and use the authorized API for downloads.

- Fine-grained buckets: every new upload sends the `private` predefined ACL. A public default object ACL cannot carry over to that upload.
- Uniform buckets: uploads send no object ACL. The copier first reads the bucket settings and IAM policy and rejects public members in any binding, including conditional bindings.
- Every new file uses `Cache-Control: private, no-store` and a create-only generation condition.
- Existing files and completed uploads pass the same check. Fine-grained files need a readable, non-empty ACL without `allUsers` or `allAuthenticatedUsers`. The full metadata response keeps this ACL and its source metadata together. Firebase download tokens also block readiness.
- A missing access check returns an error. Public access or a download token requires operator review. A retry does not grant or remove access.

The API identity needs bucket metadata and IAM read access. In fine-grained mode it also needs full object metadata with the object ACL. Verify these permissions in the intended environment before enabling workers. Do not add broad roles to make a failed check pass.

Use a dedicated private bucket. Public access prevention is a useful bucket guard, especially for old ACLs, but it is an explicit operator change. This source change does not enable it, change existing ACLs, remove download tokens, or clear old public caches. The copier still rejects a public object ACL when public access prevention blocks its effect. Review and repair that ACL separately before retrying the copy.

## Local evidence

From `api/`, run:

```sh
go run ./cmd/evalstorage -run
```

The command invokes the production copier and Storage SDK through a scripted HTTP transport. It has no socket transport, credentials, application configuration, or real provider calls. It checks private upload requests, uniform mode, public and unreadable access, existing-file replay, download tokens, and verification after upload. It rejects a missing opt-in or unknown arguments. This checks SDK requests and local policy; it does not establish deployed permissions, Firebase download behavior, or cache expiry.

Keep historical access review separate from these defaults. See [release safety](RELEASE-SAFETY.md) before a rollout.

## Verified source guard

The attachment adapter checks bucket scope, public bucket bindings, full object ACLs, and Firebase download-token metadata before reporting an upload as ready. The local SDK evaluation covers 17 cases and 4 refusal gates without a real network call. The integration commit is `8aa4c38`; live old-file access has a separate review procedure.
