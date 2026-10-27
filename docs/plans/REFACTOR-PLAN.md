# Modular refactor plan

Keep one Go API and one Next.js app. Use business modules with clear owners. Track delivery in [issue #1](https://github.com/velrino/classgame/issues/1) and [release PR #16](https://github.com/velrino/classgame/pull/16).

## 1. What would you do differently?

- Group code by capability: identity, classrooms, missions, learning, insights, attachments, and game.
- Keep route files and handlers small. Put workflows in focused files.
- Use short English guides in `docs/`, with one `README.md` per folder. Use UPPERCASE Markdown file names.
- Use Scalar to read the OpenAPI contract. Keep the Product tour and original architecture SVG unchanged.
- Keep one Go module and explicit manual dependency wiring. Add a layer only when it helps.

## 2. Coupling

- Let consumers define small Go interfaces. Keep provider SDKs at the edges.
- Use frontend public exports, adapters, hooks, and UI. Keep browser storage and HTTP outside pure learning rules.
- Keep required Firestore transactions. A module move must not split an atomic write.
- Use a neutral job port. The queue handles delivery; the learning module checks current state.

## 3. DRY

- Reuse HTTP mechanics, auth parsing, and focused policies when their meaning is the same.
- Use one evidence queue and stable IDs for answer, reasoning, and feedback events.
- Use Python for evaluation datasets and reports. Call real Go policies through a small bridge.
- Avoid generic `utils`, `common`, and large shared models. Small local repetition can keep modules independent.

## 4. Vulnerabilities

- Enforce identity provider policy and resource access separately.
- Derive scores from accepted server events. Client checkpoints only help resume the game.
- Pin each request to its mission revision and attempt. Reject changed requests that reuse an event ID.
- Keep teacher notes private. Use student-safe responses and authorized attachment downloads.
- Bound model work, retries, payloads, and queue capacity. Use claim tokens to reject late worker results.
- Keep jobs safe across instances. A retry must not repeat an accepted business effect.

## Release checks

Run the existing Go and web checks, offline policy evaluations, dependency scans, and affected local flows. Check links, file names, the Product tour, and the original SVG. Use the [release workflow](../processes/RELEASE-WORKFLOW.md); a merge does not deploy the app.

## Completed access work and future deployment

The [validation record](../operations/RELEASE-VALIDATION.md) now records measured local queries, browser recovery, shared budgets, retention checks, and separately approved cloud access restrictions. The four old private files now have a read-only exact-path grant for the existing API identity. All 12 private reads pass and anonymous reads remain denied. No app deployment, file deletion, or shared-data bootstrap ran. Future hosted setup follows [release safety](../operations/RELEASE-SAFETY.md).
