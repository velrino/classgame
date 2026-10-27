# ClassGame — contributor router

One Go API and one Next.js application. Read the nearest package guide and the [documentation task index](docs/README.md).

| Work | Read first |
| --- | --- |
| API, storage, workers, generation | [API guide](api/AGENTS.md) |
| Dashboard, enrollment, 3D game | [Web guide](web/AGENTS.md) |
| Parallel agents or future automation | [Agent contract](agents/AGENTS.md) |
| Architecture and boundaries | [Architecture](docs/architecture/README.md) |
| Documentation | [Documentation standard](docs/conventions/DOCUMENTATION.md) |
| Review and release | [Processes](docs/processes/README.md) |
| AI safeguards and evaluation | [AI](docs/ai/README.md) |

## Working rules

- Do not use test-driven development (TDD).
- Do not add new tests unless the user explicitly asks for them.
- Existing tests may be run after implementation as validation.
- Work within the requested scope. Preserve unrelated edits and assign explicit file ownership for parallel work.
- Keep credentials, bearer tokens, student content, and worksheet text out of Git, logs, and review artifacts.
- Use module contracts rather than private imports or another module's persistence documents. Preserve wire contracts and required transactions.
- Write repository artifacts in English. Use Conventional Commit subjects and only `velrino <velrino@gmail.com>` as author and committer, with no co-author trailers.
- Follow the authorized [feature → release → master workflow](docs/processes/RELEASE-WORKFLOW.md). Report observed results accurately; a merge is not a deployment.

Detailed guidance has one canonical home under `docs/`. Package READMEs remain short entry points; asset provenance stays beside the assets.
