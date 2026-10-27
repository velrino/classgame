# Cloud Run operations

The previous hosted deployment is offline; see [hosting history](hosting-history.md). Local startup does not recreate it. This guide describes the infrastructure contract to review before an intentional deployment.

## Build inputs

Run builds from the repository root, where both application directories are available:

```bash
gcloud builds submit --config api/infra/cloud-run/production/cloudbuild.yaml .
gcloud builds submit --config web/infra/cloud-run/production/cloudbuild.yaml .
```

The web build requires its public Firebase configuration, explicit API base URL, and optional Uploadcare public key through the substitutions declared in [its build configuration](../../web/infra/cloud-run/production/cloudbuild.yaml). Keep private values in runtime secret configuration; do not embed them in the browser build.

Review project, region, service names, repository paths, identities, and substitutions in [API](../../api/infra/cloud-run/production/cloudbuild.yaml) and web configurations before invoking these commands. Deployment scripts retain operational assumptions that must match the intended environment.

## Runtime requirements

- Firestore, Firebase Auth, Storage, and the reviewed [browser rules](../../firestore.rules).
- Redis with scoped credentials for quotas and dependency checks.
- Runtime credentials through Secret Manager or workload identity, with only necessary permissions.
- Vertex AI or another supported GenAI backend when live generation is enabled.
- Cloud Tasks queue, dispatcher identity, worker URLs, and matching OIDC audience when background delivery is enabled.
- Artifact Registry and Cloud Build permissions for container publication.

Task workers validate a Google-signed token and the expected service-account identity. A task-specific header is not authentication. Preserve deterministic task IDs, classify retryable/permanent failures, and retain referenced published revisions.

Attachment access must remain authorized after storage copying; new object defaults do not change existing public objects. Review existing access as an explicit migration. Keep generation able to read authorized attachments without publishing them.

## Verification

After deployment, check health, `/docs`, `/openapi.json`, web routes, identity policy, and an authenticated teacher/student workflow. Verify actual deployed rules, Storage policy, queue delivery, and IAM separately. A successful build or healthy dependency response is not evidence that those controls or gameplay were exercised.

Use identifiers to correlate requests, attempts, missions, and tasks. Keep worksheet text, student content, credentials, and bearer tokens out of logs. Separate operational metrics from product events; measure latency, reads, retries, fallback frequency, and projection lag.

If durable dispatch is required, design an outbox/reconciler with documented failure recovery. Do not replace atomic publication with independent writes just to introduce events.
