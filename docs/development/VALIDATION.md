# Validation

Run checks appropriate to the affected boundary. Existing tests may run; repository instructions require an explicit request before adding tests.

From `web/`:

```bash
npx tsc --noEmit
npm run lint
npm run build
npm audit --json
```

From `api/`:

```bash
go test ./...
go build ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

From the repository root, run the requested offline suites:

```bash
python3 evals/run.py
python3 evals/run.py --dataset evals/platform/offline-evidence/cases.json
```

Python uses production Go and TypeScript policies. These checks use synthetic data and no real model calls. Optional local Redis and Firestore checks are described in the [eval guide](../../evals/README.md). They are separate from shared cloud services.

Run production web builds separately from the active development server: both use the default Next.js build directory. An audit exit code needs advisory triage; a passing build does not resolve vulnerabilities.

## Runtime checks

```bash
curl --fail http://localhost:4004/
curl --fail http://localhost:4004/openapi.json
curl --fail http://localhost:4004/docs
curl --fail http://localhost:3000/login
curl --fail http://localhost:3000/home
```

Check affected workflows with development data:

- Teacher sign-in, class creation, mission preview streaming, draft conflicts, and publication.
- Student join, mission selection, incorrect answer, explanation, feedback, completion, and checkpoint recovery after reload.
- Attempts stay pinned after a newer revision is published; replayed requests do not duplicate progress.
- Other teachers and students cannot access protected classroom data. Student payloads exclude private observations and administrative fields.
- Auth/route changes cancel requests, unsubscribe listeners, and dispose Three.js resources.
- Duplicate/delayed tasks preserve finalization; unavailable providers select the documented fallback.

For documentation changes, verify local Markdown and HTML links, legacy anchors, and image references. The root Product tour and original architecture SVG are protected content: preserve their text, links, images, and bytes.

Record what ran and its result in the PR. Distinguish build success, local smoke checks, authenticated workflow checks, deployment, and physical-device validation.
