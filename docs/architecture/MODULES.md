# Module boundaries

Use business capabilities as code boundaries. Keep one Go module, one web app, and no import cycles. Move one working use case at a time.

## Go

```text
api/cmd/api/                   # Composition, HTTP registration, lifecycle
api/internal/
  config/                      # Environment parsing and validation
  modules/<capability>/
    domain/                    # Pure models, invariants, errors
    application/               # Use cases, commands, consumer-owned ports
    adapters/
      http/                    # Decode, validate shape, invoke, project
      firestore/               # Documents, mappings, queries, transactions
      genai/                   # Capability-specific generation
      tasks/                   # Enqueue and worker integration
  platform/                    # Provider clients and connection lifecycle
```

The tree shows optional layers, not folders to create now. Current capabilities use focused files in one package. Create a subpackage when it gives a useful boundary.

- Dependency direction is HTTP adapter → application → domain. Provider adapters implement application-owned ports; composition wires them.
- Domain policy uses business types and standard-library primitives. Echo, Firebase, Redis, GenAI, and task SDKs stay at the edges.
- Define small reader/writer interfaces beside their consumers. Constructors receive explicit dependencies and reject invalid required combinations.
- Keep HTTP DTOs, Firestore documents, and domain types distinct where their meanings differ. Preserve the existing `snake_case` wire contract through mappers.
- Modules use explicit contracts rather than another module's collection names or persistence documents. Preserve cross-document transactions such as publication as an explicit operation.
- Use synchronous calls for synchronous work. Add events or an outbox only for a documented delivery requirement, with idempotency and recovery behavior.

### Shared Go code

Share a package only when consumers share its meaning and an owner is clear. Prefer focused names such as `ratelimit`, `contentpolicy`, or `timeutil`; keep provider packages separate from business policy. Technical shared packages must not import business modules.

The technical [rate-limit capability](RATE-LIMITING.md) lives under `internal/platform/ratelimit`, with standard-library core policy and separate `redis/` and `http/` adapters. Business quota rules stay outside that technical package.

Keep application-only packages under `internal/`; `pkg/` has no special export behavior. Do not create extra `go.mod` files, universal repositories, service locators, or wrappers around every standard-library call. Map distinct domain models rather than forcing them into a large shared object.

## Frontend

```text
web/src/modules/<capability>/
  index.ts                     # Supported public exports
  <Capability>Root.tsx          # Optional mountable composition
  adapters/
    types/                     # Module-owned host contracts
    context/                   # Optional provider wiring
    hooks/                     # Typed capability access
  host/                        # Application integration bridge
  data/
    http/                      # Transport and response parsing
    realtime/                  # Subscriptions and cleanup
    storage/                   # Browser persistence
  features/<use-case>/
    index.ts
    api/                       # Feature endpoint operations
    queries/                   # Queries, mutations, keys when needed
    hooks/                     # Workflow controllers
    ui/                        # Feature views
    schemas/                   # Boundary validation
    types/                     # Feature contracts
  hooks/                       # Cross-feature composition
  ui/
    base/                      # Pure primitives
    shared/                    # Pure composed views
    layout/                    # Module chrome
  domain/                      # Pure policies and transitions
  configs/                     # Module constants
  types/                       # Contracts shared within this module
```

This is a template, not an empty-folder requirement. Add a root/provider only for a mountable surface that needs it. Keep game scene, controls, audio, and avatars behind its engine/world boundary.

- Consumers use a module's `index.ts` or intentional `host` entrypoint. Cross-feature imports use feature public exports, without deep imports.
- The host supplies stable identity, token, navigation, configuration, and telemetry primitives. Module core never imports its host, route files, or initialized SDK singletons.
- Modules own endpoint operations, schemas, queries, and workflows. Avoid a host callback for every business operation.
- UI primitives take props and emit intent. Controllers compose data, domain policy, loading/error state, and cancellation. Firebase and persistence details stay in data adapters.
- Next.js routes/layouts remain in `src/app`. Preserve server/client and lazy-loading boundaries when mounting modules.
- Shared UI and transport leaves do not import business modules. Promote code only when consumers share its semantics.
- Reuse application providers and cache ownership. A `queries/` folder does not require a new library or a cache per module.
- Enforce imports for both aliases and resolved relative paths. Check public exports for cycles and avoid eagerly importing the game engine through a barrel.

Keep mission preview SSE separate from JSON requests. Preserve session-scoped invalidation, checkpoint recovery order, unsubscribe behavior, and resource disposal throughout migration.
