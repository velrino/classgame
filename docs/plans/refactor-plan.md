# Modular architecture implementation plan

Scope: the ClassGame Go API, Next.js application, access rules, infrastructure references, documentation, and assets. Keep a modular monolith; preserve teacher/student workflows while moving one use case at a time.

Implementation is tracked through [repository issues](https://github.com/velrino/classgame/issues) and feature PRs into `release/modular-architecture`. This document defines acceptance; it does not claim every item is complete.

## 1. What would you do differently?

- Give identity, classrooms, missions, learning, tutoring, insights, attachments, and game rendering explicit ownership.
- Split large workflow services and components by responsibility; keep routes and handlers thin.
- Use business names and truthful provider-policy errors. Deterministic production tutor behavior is policy, not disposable mock code.
- Make `docs/` canonical, root/component READMEs concise entrypoints, and asset notes contextual.
- Retain one `api/go.mod`; use the [Go and frontend module anatomy](../architecture/modules.md) without adding empty layers.

## 2. Coupling

- Replace concrete repositories, silent capability assertions, and post-construction setters with explicit consumer-owned ports and validated dependencies.
- Keep Firestore documents, SDK clients, route state, and analytics singletons outside domain policy. Frontend modules expose public entrypoints and inject small host primitives through adapters.
- Separate game recovery, synchronization, tutor requests, and the world bridge; preserve session cancellation and resource disposal.
- Document publication's atomic cross-document boundary. Separate storage ownership without splitting required transactions into independent calls.

## 3. DRY

- Centralize bearer authentication and transport error mechanics while retaining use-case authorization.
- Consolidate shared identity contracts, frontend request/JSON parsing, mission rules, and generation metadata only where their semantics match.
- Keep mission preview SSE separate from JSON endpoints and preserve server validation independently of browser checks.
- Avoid universal repositories, catch-all utilities, or shared persistence models. A shared Go package needs a clear owner, focused purpose, minimal dependencies, and multiple justified consumers.

## 4. Vulnerabilities

Prioritize demonstrated trust-boundary failures before structural moves:

- Enforce configured identity providers independently of classroom authorization.
- Deliver private attachments through authorized access; preserve generation reads and migrate existing public objects separately.
- Separate resume checkpoints from authoritative answer events and server-derived completion.
- Replace student reads of administrative class documents with minimal projections. Backfill and switch consumers before restricting old access.
- Triage current dependency advisories for reachability and compatible patches. Avoid forced fixes or unrelated major-version downgrades.
- Define environment-specific CORS, multi-instance rate limits, class-code lifecycle, task OIDC policy, and AI input/output controls. Treat unresolved product decisions and deployment checks as such, rather than confirmed exploits.

## Delivery order

| Phase | Work | Acceptance |
| --- | --- | --- |
| Baseline | Record routes, data ownership, configuration, protected media, and representative behavior. | Reproducible setup and measured baseline; original Product tour and architecture SVG preserved. |
| Security | Provider policy, private attachments, authoritative progress, minimal projections, dependency triage. | Positive/negative access checks; migration and compatibility evidence. |
| Backend | Explicit construction, ports, shared contracts, capability extraction. | No silent capability detection or provider documents crossing business boundaries; routes and transactions preserved. |
| Frontend | Public module surfaces, adapters, data boundaries, controllers, pure UI. | No deep cross-module imports; recovery, SSE, subscriptions, and game lifecycle preserved. |
| Scale/reliability | Bounded insights queries, rebuildable projections, deliberate task delivery. | Stable cursors/limits, visible freshness, replay safety, and documented recovery. |
| Organization | Canonical docs, naming, asset and infrastructure references, confirmed unused code. | Working links, legacy anchors, build contexts, runtime asset URLs, and scoped instructions. |

Keep security/behavior changes separate from mechanical moves. Use additive projection migrations and small reversible commits. An outbox, event bus, new cache, or service split needs a demonstrated requirement.

Measure query reads, latency, retries, fallback rates, and projection lag with request/attempt/task correlation and no sensitive content. Validate affected flows and existing checks using [the validation guide](../development/validation.md). Repository instructions prohibit adding tests unless explicitly requested.
