# Release workflow

Integrate authorized work in this order: **feature branch → release branch → `master`**. The modular refactor uses `release/modular-architecture`; feature PRs target that branch, not `master`.

## Prepare a feature

1. Start from the current release branch. Use separate worktrees and explicit owned paths for parallel tasks; see [agent workspaces](../ai/agent-workspaces.md).
2. Make a scoped change and update canonical guidance when behavior/contracts change. Preserve unrelated working-tree edits.
3. Run relevant [validation](../development/validation.md). Inspect the staged diff, credentials, client compatibility, and migration/rollback implications.
4. Commit and push when authorized. Use Conventional Commit subjects such as `refactor(web): split mission editing`. Use only `velrino <velrino@gmail.com>` as author and committer, without co-author trailers. Verify the intended local SHA against the remote branch.
5. Open a feature PR against the release branch using the [four review lenses](code-review.md). Record checks, limitations, and required operational work.

`feature/`, `fix/`, and `refactor/` describe the work; the release target defines the integration boundary. Do not force-push or rewrite shared history without explicit task authorization.

## Integrate and promote

The coordinator reviews and merges feature PRs in dependency order. Resolve conflicts in the feature branch and validate the integrated release. Independent branches passing does not establish combined compatibility.

Open the final release PR to `master` after required checks/reviews. Follow the task's merge/publishing authorization. Verify the resulting remote commit and distinguish pushed, merged, deployed, and authenticated-workflow validation.

Git integration does not deploy infrastructure. Use [operations](../operations/README.md) for compatibility, access-policy changes, recovery, and environment checks. Historical screenshots and hosting records do not establish a running deployment.
