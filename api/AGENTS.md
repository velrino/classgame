# API — package guide

The Go API owns classroom authorization, missions, student learning, generation, attachments, and task handlers. Read the [root rules](../AGENTS.md) first.

| Task | Canonical guide |
| --- | --- |
| Start or configure | [Local setup](../docs/development/LOCAL-SETUP.md), [configuration](../docs/reference/CONFIGURATION.md) |
| Change a use case or dependency | [Module boundaries](../docs/architecture/MODULES.md) |
| Change routes or stored projections | [API](../docs/reference/API.md), [data model](../docs/reference/DATA-MODEL.md) |
| Change generation or context | [AI safeguards](../docs/architecture/AI-QUALITY.md), [AI index](../docs/ai/README.md) |
| Validate or operate | [Validation](../docs/development/VALIDATION.md), [operations](../docs/operations/README.md) |

- Keep one [go.mod](go.mod). Composition starts at [cmd/api](cmd/api/main.go).
- Keep SDKs and persistence mappings at the edges. Define narrow ports beside their consumers; keep policy independent of provider state.
- Resolve identity and resource access separately. Exclude teacher-only observations from student projections and provider inputs that do not need them.
- Preserve `snake_case` payloads, pinned revisions, publication transactions, and idempotent tasks. External calls stay outside retryable Firestore transactions.
- Local startup does not authorize shared-data bootstrap, reset scripts, rule deployment, or hosted-resource recreation. Use the current task's authorization.

Run relevant existing checks here: `go test ./...`, `go build ./...`, `go vet ./...`. Follow the root policy before adding tests, and report workflow/environment checks separately.
