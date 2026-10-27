# Web — package guide

Next.js teacher dashboard and Three.js student classroom. Read the [root rules](../AGENTS.md) and the installed, version-matched Next.js guides below.

| Task | Canonical guide |
| --- | --- |
| Start or configure | [Local setup](../docs/development/LOCAL-SETUP.md), [configuration](../docs/reference/CONFIGURATION.md) |
| Change module ownership | [Module boundaries](../docs/architecture/MODULES.md) |
| Change API behavior | [API contract](../docs/reference/API.md) |
| Check a change | [Validation](../docs/development/VALIDATION.md) |
| Review or integrate | [Code review](../docs/processes/CODE-REVIEW.md), [release workflow](../docs/processes/RELEASE-WORKFLOW.md) |

- Routes/layouts stay in [src/app](src/app). Business behavior belongs to [src/modules](src/modules), with intentional public surfaces.
- Put HTTP, realtime schemas, browser recovery, and vendor access behind module adapters. Keep UI intent separate from workflow controllers and pure policies.
- Preserve server/client boundaries, revision pinning, checkpoint ordering, subscription cancellation, and Three.js disposal.
- Every `NEXT_PUBLIC_*` value is browser-visible. Keep private credentials and teacher-only context server-side.
- Use scripts from [package.json](package.json). Run production builds separately from an active development checkout; both use the default build directory.

Run relevant existing checks here: `npx tsc --noEmit`, `npm run lint`, `npm run build`, and `npm audit --json`. Follow the root rule before adding tests; report browser/device workflows separately.

<!-- BEGIN:nextjs-agent-rules -->

## This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->
