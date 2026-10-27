# ClassGame — contributor router

One Next.js app and one Go module. Firestore owns durable state. Optional Asynq workers use a dedicated Redis broker; quotas and shared AI limits use separate Redis. Mia tutoring stays synchronous.

Read the nearest package guide and the [documentation task index](docs/README.md).

| Work | Read first |
| --- | --- |
| API, storage, workers, generation | [API guide](api/AGENTS.md) |
| Dashboard, enrollment, 3D game | [Web guide](web/AGENTS.md) |
| Evaluation tools and datasets | [Evaluation guide](evals/AGENTS.md) |
| Parallel tasks and handoffs | [Agent contract](agents/AGENTS.md) |
| Architecture and module boundaries | [Architecture](docs/architecture/README.md) |
| Architecture image | [PNG renderer guide](assets/diagrams/architecture/README.md) |
| Documentation | [Documentation standard](docs/conventions/DOCUMENTATION.md) |
| Review and release | [Processes](docs/processes/README.md) |
| AI safeguards | [AI](docs/ai/README.md) |

## Working rules

- Do not use test-driven development (TDD). Do not add new tests unless the user explicitly asks. Run relevant existing checks after a change.
- Stay within the task scope. Preserve unrelated edits and user processes. Assign owned paths for parallel work.
- Keep credentials, bearer tokens, student content, and worksheet text out of Git, logs, fixtures, and review artifacts. Use synthetic data for checks.
- Use module public contracts, small consumer-owned interfaces, and explicit wiring. Keep shared code focused on one purpose; follow the module guide for dependency rules.
- Write in English B1. Use UPPERCASE Markdown file names and keep canonical human guidance under `docs/`.
- Preserve the Product tour and its media unless the user requests a change. Architecture output is PNG only; edit [architecture.py](assets/diagrams/architecture/architecture.py), regenerate, and sync the dashboard copy.
- Keep work local by default. Local setup does not authorize cloud deployment, shared-data resets, or access changes. Reuse existing task authorization.
- Use Conventional Commit subjects and only `velrino <velrino@gmail.com>` as author and committer, with no co-author trailers.
- Track requested work in GitHub issues and follow the authorized [feature → release → master workflow](docs/processes/RELEASE-WORKFLOW.md). Use the [four review lenses](docs/processes/CODE-REVIEW.md) in reviews and PRs.
- Close issues after verified completion. Remove only merged task branches/worktrees. State actual checks and limits: a build is not a user journey, and a merge is not a deployment.

Keep detailed rules in their canonical guides. Package READMEs stay short; asset credits remain beside their files.
