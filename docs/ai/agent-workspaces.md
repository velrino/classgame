# Agent workspaces

Current coding work runs as scoped tasks in Git worktrees. Mia is application behavior in the API, not a repository maintenance bot. The checked-in `agents/` contract does not establish a deployed autonomous service.

## Start and isolate

The coordinator supplies an objective, owned paths, starting release revision, branch/worktree, and expected handoff. Read the root router and nearest package guide before editing. Seek missing context only when it prevents useful authorized progress.

Give each task its own checkout and runtime files. Coordinate shared-file edits; do not overwrite another task or claim its port. Worktrees share Git objects and may share a machine, so they do not isolate files, processes, or credentials.

Use synthetic or appropriately scoped development data for checks. Keep private student data, worksheets, credentials, and tokens out of prompts, committed fixtures, logs, and PR artifacts unless the task requires an authorized handling path.

## Tools and authority

Use capabilities actually available in the session. Documentation does not install a tool, create a credential, or authorize a remote mutation. Continue within existing task authorization. Cloud changes, data resets, publishing, and external messages are separate actions when outside that scope.

Future standalone automation follows the [host/sandbox credential contract](../../agents/AGENTS.md). Keep that design conditional until its implementation, identity, checks, and operational guide exist.

## Hand off observed results

Return the branch/SHA, owned paths, resulting behavior, validation commands/results, unresolved risks, and processes left running. A build is not a tested user journey; a merged PR is not a deployed release.

The coordinator owns integration through the [release workflow](../processes/release-workflow.md). Save durable guidance under `docs/`; do not turn transient task assignments into claims about permanent agents.
